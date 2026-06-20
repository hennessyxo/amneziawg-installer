// Package sysstat samples host resource usage (CPU, load, memory, disk, uptime)
// from /proc for the web panel's server overview. All readers are tolerant: a
// missing or unreadable source yields a zero value rather than an error, so a
// partial or non-Linux environment still renders something sensible.
package sysstat

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

// procRoot is the mount point for the proc filesystem (overridable in tests).
var procRoot = "/proc"

// Stat is a point-in-time snapshot of host resource usage.
type Stat struct {
	CPUPercent     float64 // busy fraction 0..100 since the previous sample
	Load1          float64
	Load5          float64
	Load15         float64
	MemUsedBytes   uint64
	MemTotalBytes  uint64
	DiskUsedBytes  uint64
	DiskTotalBytes uint64
	UptimeSeconds  int64
}

// Collector samples host stats over time. CPU% needs two /proc/stat reads, so a
// Collector keeps the previous totals between calls. Safe for concurrent use.
type Collector struct {
	mu        sync.Mutex
	lastIdle  uint64
	lastTotal uint64
	primed    bool
}

// NewCollector returns a ready Collector.
func NewCollector() *Collector { return &Collector{} }

// Sample returns a current snapshot. CPUPercent is the busy fraction since the
// previous Sample call (0 on the first call, before a baseline exists).
func (c *Collector) Sample() Stat {
	st := Stat{}
	st.Load1, st.Load5, st.Load15 = parseLoad(readFile(procRoot + "/loadavg"))
	st.MemUsedBytes, st.MemTotalBytes = parseMem(readFile(procRoot + "/meminfo"))
	st.UptimeSeconds = parseUptime(readFile(procRoot + "/uptime"))
	st.DiskUsedBytes, st.DiskTotalBytes = readDisk("/")
	st.CPUPercent = c.cpuPercent(readFile(procRoot + "/stat"))
	return st
}

// cpuPercent computes the busy fraction since the previous /proc/stat read.
func (c *Collector) cpuPercent(stat string) float64 {
	idle, total := parseCPU(stat)
	if total == 0 {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.primed {
		c.lastIdle, c.lastTotal, c.primed = idle, total, true
		return 0
	}
	// Guard against counters going backwards (e.g. CPU hotplug / wraparound).
	if idle < c.lastIdle || total <= c.lastTotal {
		c.lastIdle, c.lastTotal = idle, total
		return 0
	}
	dIdle := idle - c.lastIdle
	dTotal := total - c.lastTotal
	c.lastIdle, c.lastTotal = idle, total
	busy := float64(dTotal-dIdle) / float64(dTotal) * 100
	if busy < 0 {
		busy = 0
	}
	if busy > 100 {
		busy = 100
	}
	return busy
}

// parseLoad reads the first three floats of /proc/loadavg.
func parseLoad(s string) (l1, l5, l15 float64) {
	f := strings.Fields(s)
	if len(f) >= 3 {
		l1 = atof(f[0])
		l5 = atof(f[1])
		l15 = atof(f[2])
	}
	return
}

// parseMem reads MemTotal and MemAvailable (kB) from /proc/meminfo and returns
// used and total in bytes. Used = total - available.
func parseMem(s string) (used, total uint64) {
	var avail uint64
	for _, line := range strings.Split(s, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		kb := atou(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "kB")))
		switch k {
		case "MemTotal":
			total = kb * 1024
		case "MemAvailable":
			avail = kb * 1024
		}
	}
	if total > avail {
		used = total - avail
	}
	return
}

// parseUptime reads the first field (seconds) of /proc/uptime.
func parseUptime(s string) int64 {
	f := strings.Fields(s)
	if len(f) == 0 {
		return 0
	}
	return int64(atof(f[0]))
}

// parseCPU reads the aggregate "cpu" line of /proc/stat and returns idle jiffies
// (idle+iowait) and the total across all fields.
func parseCPU(s string) (idle, total uint64) {
	for _, line := range strings.Split(s, "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		f := strings.Fields(line)[1:] // user nice system idle iowait irq softirq steal ...
		for i, v := range f {
			n := atou(v)
			total += n
			if i == 3 || i == 4 { // idle, iowait
				idle += n
			}
		}
		return
	}
	return
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func atof(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return f
}

func atou(s string) uint64 {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
