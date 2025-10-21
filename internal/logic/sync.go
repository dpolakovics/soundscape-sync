package logic

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2/widget"
	"github.com/dhowden/tag"
	ff "github.com/dpolakovics/soundscape-sync/internal/ffmpeg"
)

func getAudioFiles(folder string) ([]string, error) {
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return nil, err
	}

	var audioFiles []string
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".mp3") ||
			strings.HasSuffix(file.Name(), ".wav") ||
			strings.HasSuffix(file.Name(), ".m4b") ||
			strings.HasSuffix(file.Name(), ".mp4")) {
			audioFiles = append(audioFiles, filepath.Join(folder, file.Name()))
		}
	}

	sort.Strings(audioFiles)
	return audioFiles, nil
}

func CombineFiles(folder1, folder2, outputFolder string, progress *widget.ProgressBar, soundscapeVolume float64, debug bool, statusCallback func(string), maxParallel int) error {
	soundscapeVolume = 0.5 + (soundscapeVolume / 100.0 * 0.5)

	files1, err := getAudioFiles(folder1)
	if err != nil {
		return fmt.Errorf("failed to read audio files from folder1 (%s): %w. Check folder permissions or file formats and try again.", folder1, err)
	}
	files2, err := getAudioFiles(folder2)
	if err != nil {
		return fmt.Errorf("failed to read audio files from folder2 (%s): %w. Check folder permissions or file formats and try again.", folder2, err)
	}
	if len(files1) != len(files2) {
		return fmt.Errorf("the number of audio files in the two folders does not match: folder1 has %d files, folder2 has %d files. Please ensure both folders contain the same number of audio files and that they are in the correct order before retrying.", len(files1), len(files2))
	}

	ffmpeg := ff.FFmpegPath()
	total := len(files1)
	if total == 0 {
		statusCallback("No files to process.")
		return nil
	}

	// Pre-scan metadata so we can build commands and compute totalDuration.
	type task struct {
		index        int
		ssFile       string // soundscape file (folder1)
		bookFile     string // audiobook file (folder2)
		ext          string
		duration     float64
		channelArgs  []string
		coverArtArgs []string
		output       string
	}

	var (
		tasks         = make([]task, 0, total)
		totalDuration float64
	)

	for index, file := range files1 {
		statusCallback(fmt.Sprintf("Preparing to combine file %d of %d: %s", index+1, total, filepath.Base(file)))

		channel, err := getChannelAmount(file)
		if err != nil {
			return err
		}
		duration, err := getDuration(file)
		if err != nil {
			return err
		}
		channelArguments, err := getChannelArguments(channel, soundscapeVolume)
		if err != nil {
			return err
		}

		ext := filepath.Ext(file)
		newFileName := outputFolder + "/" + filepath.Base(files2[index])
		newFileName = newFileName[:len(newFileName)-4] + "_synced" + ext

		coverArgs := []string{}
		if ext == ".mp3" || ext == ".flac" {
			coverArgs = getCoverArtArguments(file, files2[index])
		}

		tasks = append(tasks, task{
			index:        index,
			ssFile:       file,
			bookFile:     files2[index],
			ext:          ext,
			duration:     duration,
			channelArgs:  channelArguments,
			coverArtArgs: coverArgs,
			output:       newFileName,
		})
		totalDuration += duration
	}

	// Shared cancellation: abort remaining on first error.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Progress aggregation. Track latest seconds per task, sum to a global fraction.
	latestByIndex := make([]int64, total) // store microseconds as int64 per task
	var progressMu sync.Mutex
	updateProgress := func(idx int, ms int64) {
		atomic.StoreInt64(&latestByIndex[idx], ms)
		if progress != nil && totalDuration > 0 {
			// Compute aggregate fraction safely (cheap loop over slice)
			var sumUs int64
			for i := range latestByIndex {
				sumUs += atomic.LoadInt64(&latestByIndex[i])
			}
			fraction := float64(sumUs) / 1000000.0 / totalDuration
			if fraction > 1 {
				fraction = 1
			}
			progressMu.Lock()
			progress.SetValue(fraction)
			progressMu.Unlock()
		}
	}

	// Initialize progress bar to 0 at start
	if progress != nil {
		progress.SetValue(0)
	}

	// Concurrency limit.
	if maxParallel <= 0 {
		maxParallel = runtime.NumCPU()
	}
	if maxParallel < 1 {
		maxParallel = 1
	}
	sem := make(chan struct{}, maxParallel)

	// Error collection.
	var firstErrMu sync.Mutex
	var firstErr error
	setFirstErr := func(err error) {
		firstErrMu.Lock()
		defer firstErrMu.Unlock()
		if firstErr == nil {
			firstErr = err
		}
	}

	var wg sync.WaitGroup
	for _, t := range tasks {
		wg.Add(1)
		tt := t
		go func() {
			defer wg.Done()
			// Respect global cancel
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			if debug {
				log.Printf("Starting FFmpeg for %s", filepath.Base(tt.output))
			}
			statusCallback(fmt.Sprintf("Combining file %d of %d...", tt.index+1, total))

			args := []string{"-i", tt.bookFile, "-i", tt.ssFile}
			args = append(args, tt.channelArgs...)
			args = append(args, getBaseArguments()...)
			args = append(args, tt.coverArtArgs...)
			if debug {
				args = append(args, "-loglevel", "verbose")
			}
			args = append(args, tt.output)

			cmd := exec.CommandContext(ctx, ffmpeg, args...)
			cmd.SysProcAttr = getSysProcAttr()

			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				setFirstErr(fmt.Errorf("error creating FFmpeg stdout pipe (arguments: %v): %w", cmd.Args, err))
				cancel()
				return
			}

			if debug {
				log.Printf("Executing FFmpeg command: %s %s", ffmpeg, strings.Join(args, " "))
			}

			if err := cmd.Start(); err != nil {
				setFirstErr(fmt.Errorf("error starting FFmpeg command with arguments %v: %w. Stderr: %s", cmd.Args, err, stderr.String()))
				cancel()
				return
			}

			// Progress reader for this task updates only its own latest ms
			go parseProgressAggregate(tt.index, stdout, tt.duration, updateProgress)

			if err := cmd.Wait(); err != nil {
				setFirstErr(fmt.Errorf("FFmpeg encountered an error while processing file %q with arguments %v: %w. FFmpeg stderr: %s", tt.output, cmd.Args, err, stderr.String()))
				cancel()
				return
			}

			if debug && stderr.Len() > 0 {
				log.Printf("FFmpeg stderr output for %s:\n%s", filepath.Base(tt.output), stderr.String())
			}

			statusCallback(fmt.Sprintf("Finished combining file %d of %d", tt.index+1, total))
		}()
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	statusCallback("All files combined successfully!")
	if progress != nil {
		progress.SetValue(1)
	}
	return nil
}

func getChannelAmount(file string) (int, error) {
	ffprobe := ff.FFprobePath()

	cmd := exec.Command(ffprobe, "-v", "error", "-select_streams", "a:0", "-count_packets", "-show_entries", "stream=channels", "-of", "csv=p=0", file)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to determine channel amount for file %q: %w. The file might be corrupted or unsupported.", file, err)
	}
	channels, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(string(out)), ","))
	if err != nil {
		return 0, fmt.Errorf("invalid channel information retrieved for file %q: %w. Please ensure the file is a valid audio file and consider reporting this issue if it persists.", file, err)
	}
	return channels, nil
}

func getDuration(filename string) (float64, error) {
	ffprobe := ff.FFprobePath()

	cmd := exec.Command(ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filename)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to determine duration for file %q: %w. The file might be corrupted or unsupported. If issues persist, consider contacting the developer.", filename, err)
	}
	return strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
}

func parseProgress(index int, total int, progress *widget.ProgressBar, reader io.Reader, duration float64) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_ms=") {
			timeMs, _ := strconv.ParseFloat(strings.TrimPrefix(line, "out_time_ms="), 64)
			progressBase := float64(index) / float64(total)
			progressAmount := progressBase + ((timeMs / 1000000) / float64(duration) / float64(total))
			progress.SetValue(progressAmount)
		}
	}
}

func getBaseArguments() []string {
	return []string{
		"-map", "[out]",
		"-progress",
		"pipe:1",
		"-map_metadata", "1",
	}
}

func getChannelArguments(channels int, volume float64) ([]string, error) {
	switch channels {
	case 2:
		return getStereoArguments(volume), nil
	case 6:
		return get5_1Arguments(volume), nil
	case 10:
		return get7_1_2Arguments(volume), nil
	case 12:
		return get7_1_4Arguments(volume), nil
	}
	return nil, fmt.Errorf("unsupported soundscape format with %d channels. Currently only stereo (2 channels), 5.1 (6 channels), 7.1.2 (10 channels), and 7.1.4 (12 channels) are supported. Please convert your files or contact the developer for assistance.", channels)
}

func getCoverArtArguments(file1 string, file2 string) []string {
	arguments := []string{}
	// file1 is the soundscape file (ffmpeg input 1)
	// file2 is the audiobook file (ffmpeg input 0)
	if testCoverArt(file1) { // If soundscape file (input 1) has cover art
		arguments = append(arguments, "-map", "1:v", "-c:v", "copy", "-disposition:v", "attached_pic")
	} else if testCoverArt(file2) { // If audiobook file (input 0) has cover art
		arguments = append(arguments, "-map", "0:v", "-c:v", "copy", "-disposition:v", "attached_pic")
	}
	return arguments
}

func testCoverArt(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	metadata, err := tag.ReadFrom(f)
	if err != nil {
		return false
	}

	if metadata.Picture() != nil {
		return true
	}
	return false
}

// parseProgressAggregate reads lines from FFmpeg -progress pipe and updates ms for a task.
func parseProgressAggregate(taskIndex int, reader io.Reader, taskDurationSec float64, update func(idx int, ms int64)) {
	scanner := bufio.NewScanner(reader)
	var lastUs int64
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_ms=") {
			usStr := strings.TrimPrefix(line, "out_time_ms=")
			us, _ := strconv.ParseInt(usStr, 10, 64)
			// clamp to task duration (value is in microseconds)
			maxUs := int64(taskDurationSec * 1000000)
			if us > maxUs {
				us = maxUs
			}
			lastUs = us
			update(taskIndex, lastUs)
		}
	}
	// Ensure we mark completion even if final progress didn’t arrive
	update(taskIndex, int64(taskDurationSec*1000000))
}
