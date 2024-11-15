package logic

import (
  "fmt"
)

func get5_1Arguments(volume float64) []string {
      return []string{
          // best audio format so far on ios
          "-filter_complex", "[1:a]apad[a2];[0:a]volume=" + fmt.Sprintf("%f", volume / 100) + "[a1];[a1][a2]amerge=inputs=2[out]",
          "-c:a", "aac", "-b:a", "654k",
          // mp4 output
          // "-filter_complex", "[1:a]apad[a2];[0:a][a2]amerge=inputs=2,pan=5.1|c0=c0+c6|c1=c1+c7|c2=c2|c3=c3|c4=c4|c5=c5[out]",
          // "-c:a", "eac3", "-ac", "6",
      }
}
