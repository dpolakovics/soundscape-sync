package logic

import (
  "io"
  "bufio"
  "context"
  "fmt"
  "io/ioutil"
  "os"
  "os/exec"
  "path/filepath"
  "sort"
  "strconv"
  "strings"

  "fyne.io/fyne/v2/widget"
  "github.com/dhowden/tag"
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

func CombineFiles(folder1 string, folder2 string, outputFolder string, progress *widget.ProgressBar) error {
    // Get list of audio files from both folders
    files1, err := getAudioFiles(folder1)
    if err != nil {
        return err
    }
    files2, err := getAudioFiles(folder2)
    if err != nil {
        return err
    }

    if len(files1) != len(files2) {
        return fmt.Errorf("the number of audio files in the two folders must be the same")
    }

    ffmpeg, err := getFFmpegPath()
    if err != nil {
        return err
    }

    total := len(files1)

    for index, file := range files1 {
        channel , err := getChannelAmount(file)
        if err != nil {
            return err
        }
        duration, err := getDuration(file)
        if err != nil {
            return err
        }

        channelArguments, err := getChannelArguments(channel)
        if err != nil {
            return err
        }
        
        // Get output file name
        ext := filepath.Ext(file)
        newFileName := outputFolder + "/" + filepath.Base(files2[index])
        newFileName = newFileName[:len(newFileName)-4] + "_synced" + ext

        // Construct FFmpeg command
        ctx, _ := context.WithCancel(context.Background())
        arguments := []string{
            "-i", file,
            "-i", files2[index],
        }
        arguments = append(arguments, channelArguments...)
        arguments = append(arguments, getBaseArguments()...)
        arguments = append(arguments, getCoverArtArguments(file, files2[index])...)
        arguments = append(arguments, newFileName)
        cmd := exec.CommandContext(ctx, ffmpeg, arguments...)
        cmd.SysProcAttr = getSysProcAttr()
        stdout, err := cmd.StdoutPipe()
        if err != nil {
          // prepend error with text
          err = fmt.Errorf("error creating ffmpeg command: %w", err)
          return err
        }

        // Execute FFmpeg command
        if err := cmd.Start(); err != nil {
            err = fmt.Errorf("error at ffmpeg command start: %w", err)
            return err
        }

        parseProgress(index, total, progress, stdout, duration)

        if err := cmd.Wait(); err != nil {
            err = fmt.Errorf("error at ffmpeg command wait: %w", err)
            return err
        }
    }

    cleanupTemp()

    return nil
}

func getChannelAmount(file string) (int, error) {
    ffprobe, err := getFFProbePath()
    if err != nil {
        return 0, err
    }

    cmd := exec.Command(ffprobe, "-v", "error", "-select_streams", "a:0", "-count_packets", "-show_entries", "stream=channels", "-of", "csv=p=0", file)
    out, err := cmd.Output()
    if err != nil {
        return 0, err
    }
    channels, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(string(out)), ","))
    if err != nil {
        return 0, err
    }
    return channels, nil
}

func getDuration(filename string) (float64, error) {
    ffprobe, err := getFFProbePath()
    if err != nil {
        return 0, err
    }

    cmd := exec.Command(ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filename)
    out, err := cmd.Output()
    if err != nil {
        return 0, err
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

func getChannelArguments(channels int) ([]string, error) {
  switch channels {
    case 2:
      return getStereoArguments(), nil
    case 6:
      return get5_1Arguments(), nil
    case 12:
      return get7_1_4Arguments(), nil
  }
  return nil, fmt.Errorf("Currently only stereo, 5.1 and 7.1.4 Soundscapes are supported")
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
    // Open the audio file
    f, err := os.Open(filePath)
    if err != nil {
        return false
    }
    defer f.Close()

    // Use taglib to parse the audio file
    metadata, err := tag.ReadFrom(f)
    if err != nil {
      return false
    }

    // Check if there is any artwork
    if metadata.Picture() != nil {
      return true
    }
    return false
}
