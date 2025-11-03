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

// processingJob represents a single file pair to be processed
type processingJob struct {
	soundscapeFile string
	audiobookFile  string
	outputFile     string
	index          int
	fileSize       int64 // size of larger file for smart batching
}

// processingResult holds the outcome of processing a job
type processingResult struct {
	index   int
	success bool
	err     error
}

// getFileSize returns the size of a file in bytes
func getFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// getAvailableMemoryMB returns available system memory in MB
// Returns 0 if unable to determine (will disable throttling)
func getAvailableMemoryMB() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// Use Sys (total memory obtained from OS) minus Alloc (currently allocated)
	// This is a conservative estimate of available memory
	available := (m.Sys - m.Alloc) / 1024 / 1024
	return int64(available)
}

// shouldThrottleWorkers checks if we should reduce workers due to memory pressure
// Returns true if available memory is below 500MB
func shouldThrottleWorkers() bool {
	available := getAvailableMemoryMB()
	if available == 0 {
		return false // Can't determine, don't throttle
	}
	return available < 500
}

func CombineFiles(folder1 string, folder2 string, outputFolder string, progress *widget.ProgressBar, audiobookVolume float64, soundscapeVolume float64, debug bool, useAllCores bool, statusCallback func(string)) error {

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

	total := len(files1)

	// Determine number of workers based on useAllCores setting
	numWorkers := runtime.NumCPU() / 2
	if useAllCores {
		numWorkers = runtime.NumCPU()
	}
	// Ensure at least 1 worker
	if numWorkers < 1 {
		numWorkers = 1
	}
	// Don't spawn more workers than files
	if numWorkers > total {
		numWorkers = total
	}

	statusCallback(fmt.Sprintf("Processing %d files using %d workers...", total, numWorkers))

	// Phase 1: Smart batching - scan file sizes and create jobs
	jobs := make([]processingJob, total)
	for index, file := range files1 {
		size1, err1 := getFileSize(file)
		size2, err2 := getFileSize(files2[index])

		// Use the larger of the two file sizes for sorting
		fileSize := size1
		if err1 == nil && err2 == nil && size2 > size1 {
			fileSize = size2
		}

		ext := filepath.Ext(file)
		outputFile := outputFolder + "/" + filepath.Base(files2[index])
		outputFile = outputFile[:len(outputFile)-4] + "_synced" + ext

		jobs[index] = processingJob{
			soundscapeFile: file,
			audiobookFile:  files2[index],
			outputFile:     outputFile,
			index:          index,
			fileSize:       fileSize,
		}
	}

	// Calculate total size for weighted progress
	var totalSize int64
	for _, job := range jobs {
		totalSize += job.fileSize
	}

	// Sort jobs by file size (largest first) for optimal scheduling
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].fileSize > jobs[j].fileSize
	})

	// Phase 2: Set up worker pool infrastructure
	jobChan := make(chan processingJob, total)
	resultChan := make(chan processingResult, total)

	var wg sync.WaitGroup
	var progressMutex sync.Mutex
	var completedFiles int32
	var failedJobs []processingResult
	var failedJobsMutex sync.Mutex

	// Track progress per job for aggregate calculation
	jobProgress := make(map[int]float64)
	for i := 0; i < total; i++ {
		jobProgress[i] = 0.0
	}
	var totalProgress float64 // Running total to avoid iterating map

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Worker function
	worker := func(workerID int) {
		defer wg.Done()

		for job := range jobChan {
			// Check for memory pressure and back off if needed
			if shouldThrottleWorkers() {
				if debug {
					log.Printf("Worker %d: Memory pressure detected, pausing briefly", workerID)
				}
				runtime.GC() // Suggest garbage collection
			}

			result := processJob(ctx, job, total, progress, audiobookVolume, soundscapeVolume, debug, statusCallback, &progressMutex, jobProgress, &totalProgress, totalSize)
			resultChan <- result

			// Update completed count and status
			completed := atomic.AddInt32(&completedFiles, 1)

			// Thread-safe aggregate status update
			progressMutex.Lock()
			statusCallback(fmt.Sprintf("Processing files... (%d/%d completed) using %d workers", completed, total, numWorkers))
			progressMutex.Unlock()
		}
	}

	// Phase 3: Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i)
	}

	// Phase 4: Send jobs to workers
	go func() {
		for _, job := range jobs {
			jobChan <- job
		}
		close(jobChan)
	}()

	// Phase 5: Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Phase 6: Process results as they come in
	for result := range resultChan {
		if !result.success {
			failedJobsMutex.Lock()
			failedJobs = append(failedJobs, result)
			failedJobsMutex.Unlock()
		}
	}

	// Phase 7: Report final status
	successCount := int(atomic.LoadInt32(&completedFiles)) - len(failedJobs)

	if len(failedJobs) > 0 {
		// Build error message with all failures
		var errMsg strings.Builder
		errMsg.WriteString(fmt.Sprintf("Processing completed with %d successes and %d failures:\n", successCount, len(failedJobs)))
		for _, failed := range failedJobs {
			errMsg.WriteString(fmt.Sprintf("  - File %d: %v\n", failed.index+1, failed.err))
		}
		statusCallback(fmt.Sprintf("Completed with %d failures", len(failedJobs)))
		return fmt.Errorf("%s", errMsg.String())
	}

	statusCallback("All files combined successfully!")
	return nil
}

// processJob handles the processing of a single file pair
func processJob(ctx context.Context, job processingJob, total int, progress *widget.ProgressBar, audiobookVolume float64, soundscapeVolume float64, debug bool, statusCallback func(string), progressMutex *sync.Mutex, jobProgress map[int]float64, totalProgress *float64, totalSize int64) processingResult {
	result := processingResult{
		index:   job.index,
		success: false,
	}

	channel, err := getChannelAmount(job.soundscapeFile)
	if err != nil {
		result.err = err
		return result
	}

	duration, err := getDuration(job.soundscapeFile)
	if err != nil {
		result.err = err
		return result
	}

	channelArguments, err := getChannelArguments(channel, audiobookVolume, soundscapeVolume)
	if err != nil {
		result.err = err
		return result
	}

	ffmpeg := ff.FFmpegPath()
	ext := filepath.Ext(job.soundscapeFile)

	arguments := []string{
		"-i", job.audiobookFile,
		"-i", job.soundscapeFile,
	}
	arguments = append(arguments, channelArguments...)
	arguments = append(arguments, getBaseArguments()...)
	if ext == ".mp3" || ext == ".flac" {
		arguments = append(arguments, getCoverArtArguments(job.soundscapeFile, job.audiobookFile)...)
	}

	// If debug mode is enabled, add verbose logging arguments
	if debug {
		arguments = append(arguments, "-loglevel", "verbose")
	}

	arguments = append(arguments, job.outputFile)

	if debug {
		log.Printf("Executing FFmpeg command for file %d: %s %s", job.index+1, ffmpeg, strings.Join(arguments, " "))
	}

	cmd := exec.CommandContext(ctx, ffmpeg, arguments...)
	cmd.SysProcAttr = getSysProcAttr()

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		result.err = fmt.Errorf("error creating FFmpeg stdout pipe (arguments: %v): %w", cmd.Args, err)
		return result
	}

	if err := cmd.Start(); err != nil {
		result.err = fmt.Errorf("error starting FFmpeg command with arguments %v: %w. Stderr: %s", cmd.Args, err, stderr.String())
		return result
	}

	parseProgress(job.index, total, progress, stdout, duration, progressMutex, jobProgress, totalProgress, job.fileSize, totalSize)

	if err := cmd.Wait(); err != nil {
		result.err = fmt.Errorf("FFmpeg encountered an error while processing file %q with arguments %v: %w. FFmpeg stderr: %s", job.outputFile, cmd.Args, err, stderr.String())
		return result
	}

	if debug && stderr.Len() > 0 {
		log.Printf("FFmpeg stderr output for %s:\n%s", filepath.Base(job.outputFile), stderr.String())
	}

	result.success = true
	return result
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

func parseProgress(index int, total int, progress *widget.ProgressBar, reader io.Reader, duration float64, progressMutex *sync.Mutex, jobProgress map[int]float64, totalProgress *float64, fileSize int64, totalSize int64) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_ms=") {
			timeMs, err := strconv.ParseFloat(strings.TrimPrefix(line, "out_time_ms="), 64)
			if err != nil {
				continue // Skip malformed progress lines
			}

			// Calculate this job's individual progress (0.0 to 1.0)
			jobProgressValue := (timeMs / 1000000) / duration
			if jobProgressValue > 1.0 {
				jobProgressValue = 1.0
			}

			// Calculate this job's weight based on file size
			var weight float64
			if totalSize > 0 {
				weight = float64(fileSize) / float64(totalSize)
			} else {
				// Fallback to equal weighting if totalSize is 0
				weight = 1.0 / float64(total)
			}

			// Calculate new weighted progress for this job
			newValue := jobProgressValue * weight

			// Thread-safe progress bar update using delta calculation
			progressMutex.Lock()

			// Get old value and calculate delta
			oldValue := jobProgress[index]
			delta := newValue - oldValue

			// Update this job's weighted progress in the map
			jobProgress[index] = newValue

			// Update running total by delta (O(1) instead of O(n) iteration)
			*totalProgress += delta

			// Set progress bar to the running total
			progress.SetValue(*totalProgress)

			progressMutex.Unlock()
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

func getChannelArguments(channels int, audiobookVolume float64, soundscapeVolume float64) ([]string, error) {
	switch channels {
	case 2:
		return getStereoArguments(audiobookVolume, soundscapeVolume), nil
	case 6:
		return get5_1Arguments(audiobookVolume, soundscapeVolume), nil
	case 10:
		return get7_1_2Arguments(audiobookVolume, soundscapeVolume), nil
	case 12:
		return get7_1_4Arguments(audiobookVolume, soundscapeVolume), nil
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
