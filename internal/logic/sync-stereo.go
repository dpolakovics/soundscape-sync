package logic

import "fmt"

func getStereoArguments(volume float64) []string {
	return []string{
		"-filter_complex",
		fmt.Sprintf(
			"[0:a]volume=%f,apad[a2];"+
				"[1:a][a2]amerge=inputs=2,"+
				"pan=stereo|c0<c0+c2|c1<c1+c3[out]",
			volume,
		),
	}
}
