//go:build windows && amd64

package ffmpeg

import (
  ff "github.com/soundscape-sync/ffstatic-windows-amd64"
)

func FFmpegPath() string { return ff.FFmpegPath() }

func FFprobePath() string { return ff.FFprobePath() }
