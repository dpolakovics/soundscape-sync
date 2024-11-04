//go:build darwin
// +build darwin

package logic

import (
  "fmt"
  "os/exec"
  "syscall"
)

func getSysProcAttr() *syscall.SysProcAttr {
  return &syscall.SysProcAttr{}
}

func getFFProbePath() (string, error) {
  path, err := exec.LookPath("ffprobe")
  if err == nil {
      return path, nil
  }
  
  return "", fmt.Errorf("FFprobe not found")
}

func getFFmpegPath() (string, error) {
    path, err := exec.LookPath("ffmpeg")
    if err == nil {
        return path, nil
    }
    
    return "", fmt.Errorf("FFmpeg not found")
}

func cleanupTemp() {
}
