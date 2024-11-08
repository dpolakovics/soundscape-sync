//go:build darwin
// +build darwin

package ffmpeg

import (
	_ "embed"
    "fmt"
    "os"
    "runtime"

  ffarm64 "github.com/soundscape-sync/ffstatic-darwin-arm64"
  ffamd64 "github.com/soundscape-sync/ffstatic-darwin-amd64"
)

func FFmpegPath() string {
  switch runtime.GOARCH {
    case "amd64":
      return ffamd64.FFmpegPath()
    case "arm64":
      return ffarm64.FFmpegPath()
  }
  return ""
}

func FFprobePath() string {
  switch runtime.GOARCH {
    case "amd64":
      return ffamd64.FFprobePath()
    case "arm64":
      return ffarm64.FFprobePath()
  }
  return ""
}

func init() {
  switch runtime.GOARCH {
    case "amd64":
    case "arm64":
    default:
      panic(fmt.Errorf("Running on an unknown architecture: %s\n", runtime.GOARCH))
  }
}
