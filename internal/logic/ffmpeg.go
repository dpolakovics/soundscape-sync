package logic

import (
  "fmt"
  "os/exec"
  "runtime"
  ffstatic_windows_amd64 "github.com/go-ffstatic/windows-amd64"
)

func findFFmpeg() (string, error) {

    path, err := exec.LookPath("ffmpeg")
    if err == nil {
        return path, nil
    }
    
    // Add platform-specific fallback paths
    if runtime.GOOS == "windows" {
      return ffstatic_darwin_amd64.FFmpegPath(), nil
    }
    
    return "", fmt.Errorf("FFmpeg not found")
}
