//go:build darwin && arm64

package ffmpeg

import (
  ff "github.com/soundscape-sync/ffstatic-darwin-arm64"
)

func FFmpegPath() string { return ff.FFmpegPath() }

func FFprobePath() string { return ff.FFprobePath() }
