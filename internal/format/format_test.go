package format

import (
	"testing"
	"time"
)

func TestHumanBytes(t *testing.T) {
	cases := []struct {
		in   uint64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, c := range cases {
		if got := HumanBytes(c.in); got != c.want {
			t.Errorf("HumanBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHumanRate(t *testing.T) {
	if got := HumanRate(0.4); got != "0 B/s" {
		t.Errorf("HumanRate(0.4) = %q, want 0 B/s", got)
	}
	if got := HumanRate(1048576); got != "1.0 MB/s" {
		t.Errorf("HumanRate(1MB) = %q, want 1.0 MB/s", got)
	}
}

func TestAgo(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cases := []struct {
		t    time.Time
		want string
	}{
		{time.Time{}, "never"},
		{now.Add(-10 * time.Second), "10s"},
		{now.Add(-5 * time.Minute), "5m"},
		{now.Add(-2 * time.Hour), "2h"},
		{now.Add(-3 * 24 * time.Hour), "3d"},
		{now.Add(5 * time.Second), "now"},
	}
	for _, c := range cases {
		if got := Ago(c.t, now); got != c.want {
			t.Errorf("Ago(%v) = %q, want %q", c.t, got, c.want)
		}
	}
}
