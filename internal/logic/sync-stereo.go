package logic

func getStereoArguments() []string {
      return []string{
          "-filter_complex", "[1:a]apad[a2];[0:a][a2]amerge=inputs=2,pan=stereo|c0<c0+c2|c1<c1+c3[out]",
      }
}
