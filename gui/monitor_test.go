package main

import (
	"testing"
	"time"
)

func TestParseHealthActive(t *testing.T) {
	// Arrange: 90061s = 1 day 1 hour.
	out := "ACTIVE=active\nVER=amneziawg-tools v1.0\nUPTIME=90061\nCLIENTS=3\n"

	// Act
	h := parseHealth(out)

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
	h := parseHealth("ACTIVE=inactive\nVER=\nUPTIME=0\nCLIENTS=0\n")
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
		want string
	}{
		{0, "—"},
		{-5, "—"},
		{90, "1 мин"},
		{3700, "1 ч 1 мин"},
		{90061, "1 дн 1 ч"},
	}
	for _, c := range cases {
		if got := formatUptime(c.secs); got != c.want {
			t.Errorf("formatUptime(%d) = %q, want %q", c.secs, got, c.want)
		}
	}
}

func TestFormatHandshake(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		when time.Time
		want string
	}{
		{"never", time.Time{}, "—"},
		{"just now", now.Add(-30 * time.Second), "только что"},
		{"minutes", now.Add(-5 * time.Minute), "5 мин назад"},
		{"hours", now.Add(-2 * time.Hour), "2 ч назад"},
		{"days", now.Add(-72 * time.Hour), "3 дн назад"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatHandshake(c.when, now); got != c.want {
				t.Errorf("formatHandshake(%v) = %q, want %q", c.when, got, c.want)
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
