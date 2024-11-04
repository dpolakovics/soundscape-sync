//go:build linux
// +build linux

package logic

import (
  "fmt"
  "os/exec"
)

func getFFmpegPath() (string, error) {
    path, err := exec.LookPath("ffmpeg")
    if err == nil {
        return path, nil
    }
    
    return "", fmt.Errorf("FFmpeg not found")
}

func cleanupTemp() {
}
