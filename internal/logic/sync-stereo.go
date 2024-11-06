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
      newFileName := outputFolder + "/" + filepath.Base(files2[index])
      newFileName = newFileName[:len(newFileName)-4] + "_synced.mp3"
      ctx, _ := context.WithCancel(context.Background())
      arguments := []string{
          "-i", file,
          "-i", files2[index],
          "-filter_complex", "[1:a]apad[a2];[0:a][a2]amerge=inputs=2,pan=stereo|c0<c0+c2|c1<c1+c3[out]",
      }
      arguments = append(arguments, getBaseArguments()...)
      arguments = append(arguments, getCoverArtArguments(file, files2[index])...)
      arguments = append(arguments, newFileName)
      cmd := exec.CommandContext(ctx, ffmpegPath, arguments...)
      cmd.SysProcAttr = getSysProcAttr()
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

func getCoverArtFromMp3 (filename string) error {
    ffmpegPath, err := getFFmpegPath()
    if err != nil {
        return err
    }

    cmd := exec.Command(ffmpegPath, "-i", filename, "cover.jpg")
    if err := cmd.Run(); err != nil {
        return err
    }

    return nil
}
