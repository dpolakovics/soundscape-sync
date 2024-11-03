package logic

import (
  "io/ioutil"
  "path/filepath"
  "sort"
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
