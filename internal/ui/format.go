package ui

import "math"

// sparkChars are the eight block glyphs used to draw inline throughput graphs.
var sparkChars = []rune("▁▂▃▄▅▆▇█")

// Sparkline turns a series of values into a compact block-glyph graph.
// It scales to the maximum value in the series; an empty or all-zero series
// renders as flat low blocks of the requested width.
func Sparkline(values []float64, width int) string {
	if width <= 0 {
		return ""
	}
	series := lastN(values, width)

	max := 0.0
	for _, v := range series {
		if v > max {
			max = v
		}
	}

	out := make([]rune, 0, width)
	// Left-pad with the lowest glyph so the graph is right-aligned and stable.
	for i := 0; i < width-len(series); i++ {
		out = append(out, sparkChars[0])
	}
	for _, v := range series {
		out = append(out, glyphFor(v, max))
	}
	return string(out)
}

func glyphFor(v, max float64) rune {
	if max <= 0 {
		return sparkChars[0]
	}
	idx := int(math.Round(v / max * float64(len(sparkChars)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sparkChars) {
		idx = len(sparkChars) - 1
	}
	return sparkChars[idx]
}

func lastN(values []float64, n int) []float64 {
	if len(values) <= n {
		return values
	}
	return values[len(values)-n:]
}
