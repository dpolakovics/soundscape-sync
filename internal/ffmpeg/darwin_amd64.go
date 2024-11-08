//go:build darwin && amd64

package ffmpeg

import (
  ff "github.com/soundscape-sync/ffstatic-darwin-amd64"
)

func FFmpegPath() string { return ff.FFmpegPath() }

func FFprobePath() string { return ff.FFprobePath() }
