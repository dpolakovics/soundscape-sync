package logic

import (
  "io"
  "bufio"
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
  ffprobe, err := getFFProbePath()
  if err != nil {
      return err
  }
  // ffmpeg probe if first audio file in folder 1 has more than 2 channel
  audioFiles1, err := getAudioFiles(folder1)
  if err != nil {
    return err
  }
  if len(audioFiles1) == 0 {
    return fmt.Errorf("no audio files found")
  }
  cmd := exec.Command(ffprobe, "-v", "error", "-select_streams", "a:0", "-count_packets", "-show_entries", "stream=channels", "-of", "csv=p=0", audioFiles1[0])
  out, err := cmd.Output()
  if err != nil {
    return err
  }
  // out contains a pipe and another space character at the end, so we need to trim it
  channels, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(string(out)), ","))
  if err != nil {
    return err
  }
  if channels == 2 {
    return combineStereoFiles(folder1, folder2, outputFolder, progress)
  }
  if channels == 6 {
    return combineAtmosFiles(folder1, folder2, outputFolder, progress)
  }
  return fmt.Errorf("Currently only stereo and 5.1 Soundscapes are supported", channels)
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
