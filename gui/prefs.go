package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// keyringService is the namespace under which the SSH password is stored in the
// OS secret store (macOS Keychain / Windows Credential Manager / libsecret).
const keyringService = "awg-gui-ssh"

// ProfileEntry holds ONLY non-secret fields for one saved server. The password
// is deliberately absent so it can never be written to disk.
type ProfileEntry struct {
	Host         string `json:"host"`
	User         string `json:"user"`
	Label        string `json:"label"`    // optional friendly name (e.g. "VDSina DE")
	AuthMode     string `json:"authMode"` // "password" | "key"
	IdentityPath string `json:"identityPath"`
	Remember     bool   `json:"remember"`
}

// profilesDisk is the on-disk config: a list of saved servers + the last used.
type profilesDisk struct {
	Profiles []ProfileEntry `json:"profiles"`
	Last     string         `json:"last"` // "user@host"
}

// Prefs is what the frontend receives: a profile's non-secret fields plus the
// password pulled from the OS secret store (only when Remember is set).
type Prefs struct {
	Host         string `json:"host"`
	User         string `json:"user"`
	Label        string `json:"label"`
	AuthMode     string `json:"authMode"`
	IdentityPath string `json:"identityPath"`
	Remember     bool   `json:"remember"`
	Password     string `json:"password"`
}

// prefsPath returns the config file path (e.g. ~/Library/Application Support/
// awg-gui/config.json), creating the directory if needed.
func prefsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "awg-gui")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func profileKey(user, host string) string { return user + "@" + host }

// loadProfilesDisk reads saved profiles; a missing file yields an empty set.
func loadProfilesDisk() profilesDisk {
	var d profilesDisk
	path, err := prefsPath()
	if err != nil {
		return d
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return d
	}
	if err != nil {
		return d
	}
	_ = json.Unmarshal(b, &d)
	return d
}

// saveProfilesDisk writes the config (0600, never contains a password).
func saveProfilesDisk(d profilesDisk) error {
	path, err := prefsPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// upsertEntry adds or updates a profile (matched by user@host) in d and marks it
// as the last used. Pure (no IO) so it can be unit-tested.
func upsertEntry(d profilesDisk, e ProfileEntry) profilesDisk {
	key := profileKey(e.User, e.Host)
	for i := range d.Profiles {
		if profileKey(d.Profiles[i].User, d.Profiles[i].Host) == key {
			d.Profiles[i] = e
			d.Last = key
			return d
		}
	}
	d.Profiles = append(d.Profiles, e)
	d.Last = key
	return d
}

// removeEntry drops the user@host profile from d (and clears Last if it pointed
// there). Pure (no IO).
func removeEntry(d profilesDisk, user, host string) profilesDisk {
	key := profileKey(user, host)
	out := make([]ProfileEntry, 0, len(d.Profiles))
	for _, p := range d.Profiles {
		if profileKey(p.User, p.Host) != key {
			out = append(out, p)
		}
	}
	d.Profiles = out
	if d.Last == key {
		d.Last = ""
	}
	return d
}

// upsertProfile adds or updates a profile (matched by user@host) and marks it as
// the last used.
func upsertProfile(e ProfileEntry) {
	_ = saveProfilesDisk(upsertEntry(loadProfilesDisk(), e))
}

// removeProfile drops a saved profile and forgets its stored password.
func removeProfile(user, host string) {
	_ = saveProfilesDisk(removeEntry(loadProfilesDisk(), user, host))
	forgetPassword(user, host)
}

// asPrefs converts a stored profile to the frontend shape, pulling the password
// from the secret store only when Remember is set.
func (e ProfileEntry) asPrefs() Prefs {
	p := Prefs{
		Host:         e.Host,
		User:         e.User,
		Label:        e.Label,
		AuthMode:     e.AuthMode,
		IdentityPath: e.IdentityPath,
		Remember:     e.Remember,
	}
	if e.Remember && e.AuthMode != "key" && e.Host != "" {
		p.Password = loadPassword(e.User, e.Host)
	}
	return p
}

// rememberPassword stores the password in the OS secret store.
func rememberPassword(user, host, password string) error {
	return keyring.Set(keyringService, profileKey(user, host), password)
}

// loadPassword fetches a stored password; a missing entry returns "".
func loadPassword(user, host string) string {
	pw, err := keyring.Get(keyringService, profileKey(user, host))
	if err != nil {
		return ""
	}
	return pw
}

// forgetPassword removes any stored password (ignoring "not found").
func forgetPassword(user, host string) {
	_ = keyring.Delete(keyringService, profileKey(user, host))
}
