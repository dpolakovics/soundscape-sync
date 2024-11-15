package logic

import (
  "fmt"
)

func get7_1_4Arguments(volume float64) []string {
      return []string{
          "-filter_complex",
          "[1:a]apad[a2];[0:a]volume=" + fmt.Sprintf("%f", volume / 100) + "[a1];[a1][a2]amerge=inputs=2,pan=7.1.4|FL=c0+c12|FR=c1+c13|FC=c2|LFE=c3|BL=c6|BR=c7|SL=c4|SR=c5|TFL=c8|TFR=c9|TBL=c10|TBR=c11[out]",
      }
}
