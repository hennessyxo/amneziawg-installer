package ui

import (
	"fmt"
	"math"
	"time"
)

// sparkChars are the eight block glyphs used to draw inline throughput graphs.
var sparkChars = []rune("▁▂▃▄▅▆▇█")

// HumanBytes renders a byte count as a human-readable string (e.g. "1.5 GB").
func HumanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// HumanRate renders a per-second byte rate (e.g. "3.2 MB/s").
func HumanRate(bytesPerSec float64) string {
	if bytesPerSec < 1 {
		return "0 B/s"
	}
	return HumanBytes(uint64(bytesPerSec)) + "/s"
}

// Ago renders how long ago t was, compactly ("12s", "3m", "2h", "now").
// A zero time renders as "never".
func Ago(t, now time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := now.Sub(t)
	switch {
	case d < 0:
		return "now"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

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
