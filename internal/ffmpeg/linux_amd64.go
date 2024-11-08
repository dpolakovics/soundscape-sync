//go:build linux && amd64

package ffmpeg

import (
  ff "github.com/soundscape-sync/ffstatic-linux-amd64"
)

func FFmpegPath() string { return ff.FFmpegPath() }

func FFprobePath() string { return ff.FFprobePath() }
