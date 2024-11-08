//go:build linux && amd64

package ffmpeg

import (
	_ "embed"
    "fmt"
    "os"
)

//go:embed linux_amd64/ffmpeg
var ffmpeg []byte

//go:embed linux_amd64/ffprobe
var ffprobe []byte

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
	ffmpegPath, err = writeTempExec("ffmpeg_linux_amd64", ffmpeg)
	if err != nil {
		panic(fmt.Errorf("failed to write ffmpeg_linux_amd64: %v", err))
	}
	ffprobePath, err = writeTempExec("ffprobe_linux_amd64", ffprobe)
	if err != nil {
		panic(fmt.Errorf("failed to write ffprobe_linux_amd64: %v", err))
	}
}
