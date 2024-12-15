package logic

import "fmt"

func get5_1Arguments(volume float64) []string {
      return []string{
          "-filter_complex", fmt.Sprintf("[0:a]volume=%f,apad[a2];[1:a][a2]amerge=inputs=2[out]", volume),
          "-c:a", "aac", "-b:a", "654k",
      }
}
