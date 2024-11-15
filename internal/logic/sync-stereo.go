package logic

import (
  "fmt"
)

func getStereoArguments(volume float64) []string {
      return []string{
        "-filter_complex", "[1:a]apad[a2];[0:a]volume=" + fmt.Sprintf("%f", volume / 100) + "[a1];[a1][a2]amerge=inputs=2,pan=stereo|c0<c0+c2|c1<c1+c3[out]",
      }
}
