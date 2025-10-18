package logic

import "fmt"

func get5_1Arguments(audiobookVolume float64, soundscapeVolume float64) []string {
	return []string{
		"-filter_complex",
		fmt.Sprintf(
			"[0:a]volume=%f,apad[a2];"+
				"[1:a]volume=%f[a1];"+
				"[a1][a2]amerge=inputs=2[out]",
			audiobookVolume,
			soundscapeVolume,
		),
		"-c:a", "aac", "-b:a", "654k",
	}
}
