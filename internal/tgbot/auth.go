package tgbot

import (
	"encoding/json"
	"os"
	"sort"
	"sync"

	"github.com/hennessyxo/awg-suite/internal/auth"
)

// Auth gates who may use the bot. Access requires BOTH factors: the user's
// Telegram ID must be on the admin allowlist AND that user must have entered the
// access password via /auth. Passing the password is persisted so an admin does
// not have to re-enter it after a restart; being removed from the allowlist
// revokes access immediately regardless.
type Auth struct {
	mu     sync.Mutex
	admins map[int64]bool
	pwHash string         // bcrypt hash of the access password
	passed map[int64]bool // admins who have entered the password
	path   string         // JSON file persisting the passed set
}

// NewAuth builds an Auth from the admin allowlist, the access-password bcrypt
// hash and a path to persist who has passed the password (may be empty in tests).
func NewAuth(admins []int64, pwHash, path string) *Auth {
	a := &Auth{
		admins: make(map[int64]bool, len(admins)),
		pwHash: pwHash,
		passed: map[int64]bool{},
		path:   path,
	}
	for _, id := range admins {
		a.admins[id] = true
	}
	a.load()
	return a
}

// IsAdmin reports whether the id is on the allowlist.
func (a *Auth) IsAdmin(id int64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.admins[id]
}

// IsAuthorized reports whether the id may use management commands: on the
// allowlist AND has entered the password.
func (a *Auth) IsAuthorized(id int64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.admins[id] && a.passed[id]
}

// HasPassword reports whether an access password is configured.
func (a *Auth) HasPassword() bool { return a.pwHash != "" }

// TryPassword authenticates an allowlisted user with the access password. A
// non-admin can never authenticate (returns false), so the allowlist is always
// required. On success the user is remembered (and persisted).
func (a *Auth) TryPassword(id int64, pw string) bool {
	a.mu.Lock()
	admin := a.admins[id]
	a.mu.Unlock()
	if !admin || a.pwHash == "" || !auth.CheckPassword(a.pwHash, pw) {
		return false
	}
	a.mu.Lock()
	a.passed[id] = true
	a.save()
	a.mu.Unlock()
	return true
}

// load reads the persisted passed set (best-effort).
func (a *Auth) load() {
	if a.path == "" {
		return
	}
	b, err := os.ReadFile(a.path)
	if err != nil {
		return
	}
	var ids []int64
	if json.Unmarshal(b, &ids) == nil {
		for _, id := range ids {
			a.passed[id] = true
		}
	}
}

// save writes the passed set (caller holds the lock).
func (a *Auth) save() {
	if a.path == "" {
		return
	}
	ids := make([]int64, 0, len(a.passed))
	for id := range a.passed {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if b, err := json.Marshal(ids); err == nil {
		tmp := a.path + ".tmp"
		if os.WriteFile(tmp, b, 0o600) == nil {
			_ = os.Rename(tmp, a.path)
		}
	}
}
