package tgbot

import (
	"encoding/json"
	"os"
	"sort"
	"sync"

	"github.com/hennessyxo/amneziawg-installer/internal/auth"
)

// Auth gates who may use the bot. A user is allowed if their Telegram ID is a
// preset admin OR their chat authenticated with the access password via /auth.
// Authorized chats are persisted so they survive a restart.
type Auth struct {
	mu         sync.Mutex
	admins     map[int64]bool
	pwHash     string // bcrypt hash of the access password ("" = password disabled)
	authorized map[int64]bool
	path       string // JSON file persisting the authorized set
}

// NewAuth builds an Auth from preset admin IDs, an access-password bcrypt hash
// (may be empty) and a path to persist authorized chats (may be empty for tests).
func NewAuth(admins []int64, pwHash, path string) *Auth {
	a := &Auth{
		admins:     make(map[int64]bool, len(admins)),
		pwHash:     pwHash,
		authorized: map[int64]bool{},
		path:       path,
	}
	for _, id := range admins {
		a.admins[id] = true
	}
	a.load()
	return a
}

// IsAuthorized reports whether the given id may use management commands.
func (a *Auth) IsAuthorized(id int64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.admins[id] || a.authorized[id]
}

// HasPassword reports whether an access password is configured.
func (a *Auth) HasPassword() bool { return a.pwHash != "" }

// TryPassword authenticates a chat with the access password. On success the chat
// is remembered (and persisted) and true is returned.
func (a *Auth) TryPassword(id int64, pw string) bool {
	if a.pwHash == "" || !auth.CheckPassword(a.pwHash, pw) {
		return false
	}
	a.mu.Lock()
	a.authorized[id] = true
	a.save()
	a.mu.Unlock()
	return true
}

// load reads the persisted authorized set (best-effort).
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
			a.authorized[id] = true
		}
	}
}

// save writes the authorized set (caller holds the lock).
func (a *Auth) save() {
	if a.path == "" {
		return
	}
	ids := make([]int64, 0, len(a.authorized))
	for id := range a.authorized {
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
