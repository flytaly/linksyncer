package teaprogram

import (
	"fmt"

	"github.com/gookit/color"
)

func printMoves(moves map[string]string, limit int, maxWidth int) string {
	output := ""
	i := 0
	maxWidth = max(maxWidth-7, 20)
	pathSize := maxWidth / 2
	for from, to := range moves {
		output += fmt.Sprintf(" - %s -> %s\n",
			color.Cyan.Sprint(tail(from, pathSize)),
			color.Cyan.Sprint(tail(to, pathSize)))
		if i += 1; i >= limit {
			break
		}
	}
	if left := len(moves) - limit; left > 0 {
		output += fmt.Sprintf("and %d more moves...", left)

	}
	return output
}

func tail(s string, n int) string {
	if (len(s) - n) <= 0 {
		return s
	}
	r := []rune(s)
	return "..." + string(r[len(r)-(n-3):])
}
