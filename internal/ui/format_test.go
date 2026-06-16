package ui

import (
	"testing"
	"unicode/utf8"
)

func TestSparkline(t *testing.T) {
	// Width is honored even when the series is shorter (left-padded).
	s := Sparkline([]float64{1, 2, 3}, 6)
	if got := utf8.RuneCountInString(s); got != 6 {
		t.Errorf("sparkline rune width = %d, want 6", got)
	}
	// The max value maps to the tallest glyph.
	peak := Sparkline([]float64{0, 10}, 2)
	if r := []rune(peak); r[len(r)-1] != '█' {
		t.Errorf("max value should render as full block, got %q", string(r[len(r)-1]))
	}
	// Empty series with zero width is empty.
	if Sparkline(nil, 0) != "" {
		t.Errorf("zero width should be empty")
	}
}
