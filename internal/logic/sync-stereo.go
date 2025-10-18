package logic

import "fmt"

func getStereoArguments(audiobookVolume float64, soundscapeVolume float64) []string {
	return []string{
		"-filter_complex",
		fmt.Sprintf(
			"[0:a]volume=%f,apad[a2];"+
				"[1:a]volume=%f[a1];"+
				"[a1][a2]amerge=inputs=2,"+
				"pan=stereo|c0<c0+c2|c1<c1+c3[out]",
			audiobookVolume,
			soundscapeVolume,
		),
	}
}
