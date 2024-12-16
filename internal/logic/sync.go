package logic

import (
  "bufio"
  "context"
  "fmt"
  "io"
  "io/ioutil"
  "os"
  "os/exec"
  "path/filepath"
  "sort"
  "strconv"
  "strings"

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
        if !file.IsDir() && (
          strings.HasSuffix(file.Name(), ".mp3") ||
          strings.HasSuffix(file.Name(), ".wav") ||
          strings.HasSuffix(file.Name(), ".m4b") ||
          strings.HasSuffix(file.Name(), ".mp4")) {
            audioFiles = append(audioFiles, filepath.Join(folder, file.Name()))
        }
    }

    sort.Strings(audioFiles)
    return audioFiles, nil
}

func CombineFiles(folder1 string, folder2 string, outputFolder string, progress *widget.ProgressBar, soundscapeVolume float64, debug bool, statusCallback func(string)) error {
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

        ctx, _ := context.WithCancel(context.Background())
        arguments := []string{
            "-i", file,
            "-i", files2[index],
        }
        arguments = append(arguments, channelArguments...)
        arguments = append(arguments, getBaseArguments()...)
        if ext == ".mp3" || ext == ".flac" {
          arguments = append(arguments, getCoverArtArguments(file, files2[index])...)
        }

        // If debug mode is enabled, add verbose logging arguments
        if debug {
            arguments = append(arguments, "-loglevel", "verbose")
        }

        arguments = append(arguments, newFileName)

        cmd := exec.CommandContext(ctx, ffmpeg, arguments...)
        cmd.SysProcAttr = getSysProcAttr()
        stdout, err := cmd.StdoutPipe()
        if err != nil {
          err = fmt.Errorf("error creating FFmpeg command pipeline (arguments: %v): %w. Consider running with more detailed logging or contacting the developer if the problem persists.", cmd.Args, err)
          return err
        }

        statusCallback(fmt.Sprintf("Combining file %d of %d...", index+1, total))

        if err := cmd.Start(); err != nil {
            err = fmt.Errorf("error starting FFmpeg command with arguments %v: %w. Check that the input files are accessible and not corrupted. If the issue persists, contact the developer.", cmd.Args, err)
            return err
        }

        parseProgress(index, total, progress, stdout, duration)

        if err := cmd.Wait(); err != nil {
            err = fmt.Errorf("FFmpeg encountered an error while processing file %q with arguments %v: %w. Check the input files for issues or consider contacting the developer for further assistance.", newFileName, cmd.Args, err)
            return err
        }

        statusCallback(fmt.Sprintf("Finished combining file %d of %d", index+1, total))
    }

    statusCallback("All files combined successfully!")
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
  if testCoverArt(file1) {
    arguments = append(arguments, "-map", "0:v", "-c:v", "copy", "-disposition:v", "attached_pic")
  }
  if testCoverArt(file2) {
    arguments = append(arguments, "-map", "1:v", "-c:v", "copy", "-disposition:v", "attached_pic")
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
