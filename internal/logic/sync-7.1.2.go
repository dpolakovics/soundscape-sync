package logic

func get7_1_2Arguments() []string {
      return []string{
          "-filter_complex",
          "[1:a]apad[a2];[0:a][a2]amerge=inputs=2,pan=7.1.2|FL=c0+c10|FR=c1+c11|FC=c2|LFE=c3|BL=c6|BR=c7|SL=c4|SR=c5|TFL=c8|TFR=c9[out]",
      }
}
