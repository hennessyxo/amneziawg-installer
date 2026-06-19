package lifecycle

import (
	"path/filepath"
	"testing"
	"time"
)

func TestWindowUsage(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	samples := []UsageSample{
		{Date: "2026-05-20", Used: 0},
		{Date: "2026-06-13", Used: 1500},
		{Date: "2026-06-19", Used: 5000},
	}
	const usedNow = 8000

	// today: cutoff 06-19 → baseline 5000 → 3000
	if got := WindowUsage(samples, usedNow, now, 1); got != 3000 {
		t.Errorf("today = %d, want 3000", got)
	}
	// week: cutoff 06-13 → baseline 1500 → 6500
	if got := WindowUsage(samples, usedNow, now, 7); got != 6500 {
		t.Errorf("week = %d, want 6500", got)
	}
	// month: cutoff 05-21 → latest <= is 05-20 (0) → 8000
	if got := WindowUsage(samples, usedNow, now, 30); got != 8000 {
		t.Errorf("month = %d, want 8000", got)
	}
}

func TestWindowUsageEdgeCases(t *testing.T) {
	now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	if got := WindowUsage(nil, 100, now, 7); got != 0 {
		t.Errorf("no samples → %d, want 0", got)
	}
	// counter reset: usedNow below baseline → clamp to 0
	s := []UsageSample{{Date: "2026-06-19", Used: 9000}}
	if got := WindowUsage(s, 100, now, 1); got != 0 {
		t.Errorf("reset → %d, want 0", got)
	}
	// younger than the window: only today's sample, week falls back to earliest
	s = []UsageSample{{Date: "2026-06-19", Used: 2000}}
	if got := WindowUsage(s, 5000, now, 7); got != 3000 {
		t.Errorf("young client week = %d, want 3000", got)
	}
}

func TestRecordSamplesDedupAndPrune(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "store.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = st.Put(Record{Name: "a", PubKey: "A=", UsedBytes: 100})

	day1 := time.Date(2026, 6, 19, 1, 0, 0, 0, time.UTC)
	_ = st.RecordSamples(day1)
	_ = st.RecordSamples(day1) // same day → no duplicate

	r, _ := st.Get("a")
	if len(r.Samples) != 1 || r.Samples[0].Used != 100 {
		t.Fatalf("after day1: %+v, want one sample of 100", r.Samples)
	}

	// New day captures the current cumulative usage.
	_ = st.Put(Record{Name: "a", PubKey: "A=", UsedBytes: 500, Samples: r.Samples})
	_ = st.RecordSamples(day1.AddDate(0, 0, 1))
	r, _ = st.Get("a")
	if len(r.Samples) != 2 || r.Samples[1].Used != 500 {
		t.Fatalf("after day2: %+v, want two samples (…,500)", r.Samples)
	}

	// Pruning keeps at most maxSamples.
	for i := 0; i < maxSamples+10; i++ {
		_ = st.RecordSamples(day1.AddDate(0, 0, 2+i))
	}
	r, _ = st.Get("a")
	if len(r.Samples) > maxSamples {
		t.Errorf("samples = %d, want <= %d", len(r.Samples), maxSamples)
	}
}
