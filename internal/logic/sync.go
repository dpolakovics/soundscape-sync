package logic

import (
  "io"
  "bufio"
  "io/ioutil"
  "os/exec"
  "path/filepath"
  "sort"
  "strconv"
  "strings"

  "fyne.io/fyne/v2/widget"
)

func getAudioFiles(folder string) ([]string, error) {
    files, err := ioutil.ReadDir(folder)
    if err != nil {
        return nil, err
    }

    var audioFiles []string
    for _, file := range files {
        if !file.IsDir() && (strings.HasSuffix(file.Name(), ".mp3") || strings.HasSuffix(file.Name(), ".wav") || strings.HasSuffix(file.Name(), ".m4b")) {
            audioFiles = append(audioFiles, filepath.Join(folder, file.Name()))
        }
    }

    sort.Strings(audioFiles)
    return audioFiles, nil
}

func CombineFiles(folder1 string, folder2 string, outputFolder string, progress *widget.ProgressBar) error {
  return combineStereoFiles(folder1, folder2, outputFolder, progress)
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
