package main

import (
	"testing"
	"time"
)

func TestParseHealthActive(t *testing.T) {
	// Arrange: 90061s = 1 day 1 hour.
	out := "ACTIVE=active\nVER=amneziawg-tools v1.0\nUPTIME=90061\nCLIENTS=3\n"

	// Act
	h := parseHealth(out, "ru")

	// Assert
	if !h.Running {
		t.Errorf("Running = false, want true")
	}
	if h.Version != "amneziawg-tools v1.0" {
		t.Errorf("Version = %q, want %q", h.Version, "amneziawg-tools v1.0")
	}
	if h.Clients != 3 {
		t.Errorf("Clients = %d, want 3", h.Clients)
	}
	if h.Uptime != "1 дн 1 ч" {
		t.Errorf("Uptime = %q, want %q", h.Uptime, "1 дн 1 ч")
	}
}

func TestParseHealthInactiveKeepsDefaults(t *testing.T) {
	h := parseHealth("ACTIVE=inactive\nVER=\nUPTIME=0\nCLIENTS=0\n", "ru")
	if h.Running {
		t.Errorf("Running = true, want false")
	}
	if h.Version != "—" {
		t.Errorf("Version = %q, want default %q when empty", h.Version, "—")
	}
	if h.Uptime != "—" {
		t.Errorf("Uptime = %q, want %q for 0 seconds", h.Uptime, "—")
	}
}

func TestFormatUptime(t *testing.T) {
	cases := []struct {
		secs int
		lang string
		want string
	}{
		{0, "ru", "—"},
		{-5, "ru", "—"},
		{90, "ru", "1 мин"},
		{3700, "ru", "1 ч 1 мин"},
		{90061, "ru", "1 дн 1 ч"},
		{90061, "en", "1d 1h"},
		{3700, "en", "1h 1m"},
		{90, "en", "1m"},
	}
	for _, c := range cases {
		if got := formatUptime(c.secs, c.lang); got != c.want {
			t.Errorf("formatUptime(%d, %q) = %q, want %q", c.secs, c.lang, got, c.want)
		}
	}
}

func TestFormatHandshake(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		when time.Time
		lang string
		want string
	}{
		{"never", time.Time{}, "ru", "—"},
		{"just now ru", now.Add(-30 * time.Second), "ru", "только что"},
		{"minutes ru", now.Add(-5 * time.Minute), "ru", "5 мин назад"},
		{"hours ru", now.Add(-2 * time.Hour), "ru", "2 ч назад"},
		{"days ru", now.Add(-72 * time.Hour), "ru", "3 дн назад"},
		{"just now en", now.Add(-30 * time.Second), "en", "just now"},
		{"minutes en", now.Add(-5 * time.Minute), "en", "5m ago"},
		{"days en", now.Add(-72 * time.Hour), "en", "3d ago"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatHandshake(c.when, now, c.lang); got != c.want {
				t.Errorf("formatHandshake(%v, %q) = %q, want %q", c.when, c.lang, got, c.want)
			}
		})
	}
}

func TestShortKey(t *testing.T) {
	if got := shortKey("abcdefghijklmnop"); got != "abcdefgh…" {
		t.Errorf("shortKey(long) = %q, want %q", got, "abcdefgh…")
	}
	if got := shortKey("abc"); got != "abc" {
		t.Errorf("shortKey(short) = %q, want %q", got, "abc")
	}
}

func TestParseLifecycleRecords(t *testing.T) {
	// Arrange: a valid two-record store with daily samples.
	js := `[
	  {"name":"phone","used_bytes":1000,"samples":[{"date":"2026-06-20","used":200}]},
	  {"name":"laptop","used_bytes":500,"samples":[]}
	]`

	// Act
	recs := parseLifecycleRecords(js)

	// Assert
	if len(recs) != 2 {
		t.Fatalf("len = %d, want 2", len(recs))
	}
	if recs[0].Name != "phone" || recs[0].UsedBytes != 1000 {
		t.Errorf("rec0 = %+v, want phone/1000", recs[0])
	}
	if len(recs[0].Samples) != 1 || recs[0].Samples[0].Used != 200 {
		t.Errorf("rec0 samples = %+v, want one sample used=200", recs[0].Samples)
	}
}

func TestParseLifecycleRecordsEmptyOrBad(t *testing.T) {
	for _, in := range []string{"", "   ", "not json", "{}"} {
		if got := parseLifecycleRecords(in); got != nil {
			t.Errorf("parseLifecycleRecords(%q) = %+v, want nil", in, got)
		}
	}
}
