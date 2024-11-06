package logic

import (
  "context"
  "fmt"
  "os/exec"
  "path/filepath"

  "fyne.io/fyne/v2/widget"
)

func combineAtmosFiles(folder1 string, folder2 string, outputFolder string, progress *widget.ProgressBar) error {
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

      // Get extension of file
      // ext := filepath.Ext(file)

      // Get output file name
      newFileName := outputFolder + "/" + filepath.Base(files2[index])
      newFileName = newFileName[:len(newFileName)-4] + "_synced.m4v"

      // coverArtString, err := getCoverArtString(file)
      // if err != nil {
      //   return err
      // }
      

      // Construct FFmpeg command
      ctx, _ := context.WithCancel(context.Background())
      arguments := []string{
          "-i", file,
          "-i", files2[index],
          // best audio format so far on ios
          "-filter_complex", "[1:a]apad[a2];[0:a][a2]amerge=inputs=2[out]",
          "-c:a", "aac", "-b:a", "654k",
          // mp4 output
          // "-filter_complex", "[1:a]apad[a2];[0:a][a2]amerge=inputs=2,pan=5.1|c0=c0+c6|c1=c1+c7|c2=c2|c3=c3|c4=c4|c5=c5[out]",
          // "-c:a", "eac3", "-ac", "6",
      }
      arguments = append(arguments, getBaseArguments()...)
      arguments = append(arguments, getCoverArtArguments(file, files2[index])...)
      arguments = append(arguments, newFileName)
      // map channel 1 and 2 of file 2 to channel 1 and 2 of file 1 but keep all other channels of file 1
      // it will be a 6 channel atmos output file
      cmd := exec.CommandContext(ctx, ffmpegPath, arguments...)
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

    progress.SetValue(1.0)
    return nil
}


