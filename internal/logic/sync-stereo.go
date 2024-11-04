package logic

import (
  "context"
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

    ffmpegPath, err := getFFmpegPath()
    if err != nil {
        return err
    }

    total := len(files1)

    for index, file := range files1 {
      duration, err := getDuration(file)
      if err != nil {
          return err
      }
      
      // Construct FFmpeg command
      ctx, _ := context.WithCancel(context.Background())
      cmd := exec.CommandContext(ctx, ffmpegPath,
          "-i", file,
          "-i", files2[index],
          "-filter_complex", "[0:a][1:a]amerge=inputs=2,pan=stereo|c0<c0+c2|c1<c1+c3[a]",
          "-progress",
          "pipe:1",
		      "-map", "[a]",
          outputFolder + "/" + filepath.Base(file))
      stdout, err := cmd.StdoutPipe()
      if err != nil {
          return err
      }

      // Execute FFmpeg command
      if err := cmd.Start(); err != nil {
          return err
      }

      parseProgress(index, total, progress, stdout, duration)

      if err := cmd.Wait(); err != nil {
          return err
      }
    }

    cleanupTemp()

    progress.SetValue(1.0)
    return nil
}


