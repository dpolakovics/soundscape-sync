package logic

import "fmt"

func get7_1_2Arguments(volume float64) []string {
      return []string{
          "-filter_complex",
          fmt.Sprintf("[0:a]volume=%f,apad[a2];[1:a][a2]amerge=inputs=2,pan=7.1.2|FL=c0+c10|FR=c1+c11|FC=c2|LFE=c3|BL=c6|BR=c7|SL=c4|SR=c5|TFL=c8|TFR=c9[out]", volume),
      }
}
