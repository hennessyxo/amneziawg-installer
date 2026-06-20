package sysstat

import "testing"

func TestParseLoad(t *testing.T) {
	l1, l5, l15 := parseLoad("0.52 0.48 0.40 1/234 5678\n")
	if l1 != 0.52 || l5 != 0.48 || l15 != 0.40 {
		t.Errorf("parseLoad = %v %v %v", l1, l5, l15)
	}
	if a, b, c := parseLoad(""); a != 0 || b != 0 || c != 0 {
		t.Errorf("empty load should be zero, got %v %v %v", a, b, c)
	}
}

func TestParseMem(t *testing.T) {
	in := "MemTotal:        2048000 kB\nMemFree:          100000 kB\nMemAvailable:     1024000 kB\n"
	used, total := parseMem(in)
	if total != 2048000*1024 {
		t.Errorf("total = %d", total)
	}
	if used != (2048000-1024000)*1024 {
		t.Errorf("used = %d", used)
	}
}

func TestParseUptime(t *testing.T) {
	if got := parseUptime("93784.12 180000.00\n"); got != 93784 {
		t.Errorf("parseUptime = %d, want 93784", got)
	}
	if got := parseUptime(""); got != 0 {
		t.Errorf("empty uptime = %d", got)
	}
}

func TestParseCPU(t *testing.T) {
	idle, total := parseCPU("cpu  100 0 50 800 40 0 10 0 0 0\ncpu0 ...\n")
	// idle = idle(800) + iowait(40) = 840
	if idle != 840 {
		t.Errorf("idle = %d, want 840", idle)
	}
	// total = 100+0+50+800+40+0+10+0+0+0 = 1000
	if total != 1000 {
		t.Errorf("total = %d, want 1000", total)
	}
}

func TestCollectorCPUPercent(t *testing.T) {
	c := NewCollector()
	// first call primes the baseline → 0
	if got := c.cpuPercent("cpu 100 0 100 800 0 0 0 0\n"); got != 0 {
		t.Errorf("first sample should be 0, got %v", got)
	}
	// next: total goes 1000 → 1100 (+100), idle 800 → 850 (+50) → busy 50%
	if got := c.cpuPercent("cpu 150 0 100 850 0 0 0 0\n"); got != 50 {
		t.Errorf("cpuPercent = %v, want 50", got)
	}
}
