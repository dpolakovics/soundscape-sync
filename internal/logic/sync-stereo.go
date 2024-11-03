package logic

import (
  "fmt"
  "os/exec"
  "path/filepath"

  "fyne.io/fyne/v2/widget"
)

func combineStereoFiles(folder1 string, folder2 string, outputFolder string, progress *widget.ProgressBar) error {
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

    for index, file := range files1 {
      // Construct FFmpeg command
      cmd := exec.Command("ffmpeg",
          "-i", file,
          "-i", files2[index],
          "-filter_complex", "[0:a][1:a]amerge=inputs=2,pan=stereo|c0<c0+c2|c1<c1+c3[a]",
		      "-map", "[a]",
          outputFolder + "/" + filepath.Base(file))

      // Execute FFmpeg command
      err = cmd.Run()
      if err != nil {
          return err
      }
    }



    progress.SetValue(1.0)
    return nil
}
