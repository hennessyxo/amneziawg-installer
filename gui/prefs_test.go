package main

import "testing"

func TestUpsertEntryAddsUpdatesAndTracksLast(t *testing.T) {
	var d profilesDisk

	// Add first profile.
	d = upsertEntry(d, ProfileEntry{Host: "1.1.1.1", User: "root", AuthMode: "password"})
	if len(d.Profiles) != 1 {
		t.Fatalf("Profiles = %d, want 1", len(d.Profiles))
	}
	if d.Last != "root@1.1.1.1" {
		t.Errorf("Last = %q, want %q", d.Last, "root@1.1.1.1")
	}

	// Updating the same user@host must NOT add a duplicate.
	d = upsertEntry(d, ProfileEntry{Host: "1.1.1.1", User: "root", AuthMode: "key"})
	if len(d.Profiles) != 1 {
		t.Errorf("duplicate added: Profiles = %d, want 1", len(d.Profiles))
	}
	if d.Profiles[0].AuthMode != "key" {
		t.Errorf("AuthMode = %q, want updated %q", d.Profiles[0].AuthMode, "key")
	}

	// A different server is appended and becomes the last used.
	d = upsertEntry(d, ProfileEntry{Host: "2.2.2.2", User: "admin"})
	if len(d.Profiles) != 2 {
		t.Errorf("Profiles = %d, want 2", len(d.Profiles))
	}
	if d.Last != "admin@2.2.2.2" {
		t.Errorf("Last = %q, want %q", d.Last, "admin@2.2.2.2")
	}
}

func TestRemoveEntryDropsProfileAndClearsLast(t *testing.T) {
	d := profilesDisk{
		Profiles: []ProfileEntry{
			{Host: "1.1.1.1", User: "root"},
			{Host: "2.2.2.2", User: "admin"},
		},
		Last: "root@1.1.1.1",
	}

	d = removeEntry(d, "root", "1.1.1.1")

	if len(d.Profiles) != 1 {
		t.Fatalf("Profiles = %d, want 1", len(d.Profiles))
	}
	if d.Profiles[0].Host != "2.2.2.2" {
		t.Errorf("remaining Host = %q, want %q", d.Profiles[0].Host, "2.2.2.2")
	}
	if d.Last != "" {
		t.Errorf("Last = %q, want cleared after removing the last-used profile", d.Last)
	}
}

func TestRemoveEntryKeepsLastWhenOtherRemoved(t *testing.T) {
	d := profilesDisk{
		Profiles: []ProfileEntry{
			{Host: "1.1.1.1", User: "root"},
			{Host: "2.2.2.2", User: "admin"},
		},
		Last: "root@1.1.1.1",
	}
	d = removeEntry(d, "admin", "2.2.2.2")
	if d.Last != "root@1.1.1.1" {
		t.Errorf("Last = %q, want it preserved", d.Last)
	}
}

func TestAsPrefsSkipsKeychainWhenNotRemembered(t *testing.T) {
	// Remember=false must not touch the OS secret store; password stays empty.
	e := ProfileEntry{Host: "1.1.1.1", User: "root", AuthMode: "password", Remember: false}
	p := e.asPrefs()
	if p.Password != "" {
		t.Errorf("Password = %q, want empty when not remembered", p.Password)
	}
	if p.Host != "1.1.1.1" || p.User != "root" || p.AuthMode != "password" {
		t.Errorf("asPrefs copied fields wrong: %+v", p)
	}
}
