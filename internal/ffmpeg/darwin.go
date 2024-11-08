//go:build darwin
// +build darwin

package ffmpeg

import (
	_ "embed"
    "fmt"
    "os"
    "runtime"
)

//go:embed darwin_amd64/ffmpeg
var ffmpegAmd64 []byte
//go:embed darwin_amd64/ffprobe
var ffprobeAmd64 []byte

//go:embed darwin_arm64/ffmpeg
var ffmpegArm64 []byte
//go:embed darwin_arm64/ffprobe
var ffprobeArm64 []byte

func writeTempExec(pattern string, binary []byte) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer f.Close()
	_, err = f.Write(binary)
	if err != nil {
		return "", fmt.Errorf("fail to write executable: %v", err)
	}
	if err := f.Chmod(os.ModePerm); err != nil {
		return "", fmt.Errorf("fail to chmod: %v", err)
	}
	return f.Name(), nil
}

var (
	ffmpegPath  string
	ffprobePath string
)

func FFmpegPath() string { return ffmpegPath }

func FFprobePath() string { return ffprobePath }

func init() {
	var err error

  switch runtime.GOARCH {
    case "amd64":
      ffmpegPath, err = writeTempExec("ffmpeg_linux_amd64", ffmpegAmd64)
      if err != nil {
        panic(fmt.Errorf("failed to write ffmpeg_linux_amd64: %v", err))
      }
      ffprobePath, err = writeTempExec("ffprobe_linux_amd64", ffprobeAmd64)
      if err != nil {
        panic(fmt.Errorf("failed to write ffprobe_linux_amd64: %v", err))
      }
    case "arm64":
      ffmpegPath, err = writeTempExec("ffmpeg_linux_amd64", ffmpegArm64)
      if err != nil {
        panic(fmt.Errorf("failed to write ffmpeg_linux_amd64: %v", err))
      }
      ffprobePath, err = writeTempExec("ffprobe_linux_amd64", ffprobeArm64)
      if err != nil {
        panic(fmt.Errorf("failed to write ffprobe_linux_amd64: %v", err))
      }
    default:
      panic(fmt.Errorf("Running on an unknown architecture: %s\n", runtime.GOARCH))
  }
}
