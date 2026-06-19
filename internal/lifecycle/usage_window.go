package lifecycle

import "time"

const dateLayout = "2006-01-02"

// maxSamples bounds how many daily snapshots we keep per client (~5 weeks).
const maxSamples = 35

// WindowUsage returns the bytes used over the last `days` calendar days, given
// the daily samples (ascending by Date), the current cumulative usage, and now.
//
// usage = usedNow - baseline, where baseline is the cumulative usage at the start
// of the window: the latest sample dated on/before the window's first day. If no
// sample is that old, the earliest sample is used (usage since tracking began);
// with no samples at all the window is 0.
func WindowUsage(samples []UsageSample, usedNow uint64, now time.Time, days int) uint64 {
	if len(samples) == 0 || days < 1 {
		return 0
	}
	cutoff := now.AddDate(0, 0, -(days - 1)).Format(dateLayout)

	baseline := samples[0].Used // earliest, as a fallback
	for _, s := range samples {
		if s.Date <= cutoff {
			baseline = s.Used
		} else {
			break // samples are sorted ascending; no later one can match
		}
	}
	if usedNow <= baseline {
		return 0
	}
	return usedNow - baseline
}

// Today / Last7d / Last30d report usage over the respective windows.
func (r Record) Today(now time.Time) uint64  { return WindowUsage(r.Samples, r.UsedBytes, now, 1) }
func (r Record) Last7d(now time.Time) uint64 { return WindowUsage(r.Samples, r.UsedBytes, now, 7) }
func (r Record) Last30d(now time.Time) uint64 {
	return WindowUsage(r.Samples, r.UsedBytes, now, 30)
}

// RecordSamples ensures every record has a snapshot for today's date (capturing
// the cumulative UsedBytes at the first reconcile of the day) and prunes old
// samples. Called by the enforcer after usage is applied. Persists once.
func (s *Store) RecordSamples(now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	today := now.Format(dateLayout)
	changed := false
	for _, r := range s.recs {
		if appendDailySample(r, today) {
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.save()
}

// appendDailySample adds today's sample to r if absent and prunes to maxSamples.
// Returns whether r was modified.
func appendDailySample(r *Record, today string) bool {
	if n := len(r.Samples); n > 0 && r.Samples[n-1].Date == today {
		return false // already sampled today
	}
	r.Samples = append(r.Samples, UsageSample{Date: today, Used: r.UsedBytes})
	if len(r.Samples) > maxSamples {
		r.Samples = r.Samples[len(r.Samples)-maxSamples:]
	}
	return true
}
