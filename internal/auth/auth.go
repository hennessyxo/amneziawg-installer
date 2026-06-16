// Package auth provides password verification and in-memory session management
// for the web panel. Passwords are bcrypt-hashed; sessions are random opaque
// tokens carried in a Secure, HttpOnly cookie and paired with a CSRF token.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash suitable for storing on disk.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword reports whether pw matches the stored bcrypt hash.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// Session holds per-login state. CSRF guards state-changing form posts.
type Session struct {
	CSRF   string
	Expiry time.Time
}

// Store is a concurrency-safe in-memory session store.
type Store struct {
	mu       sync.Mutex
	sessions map[string]Session
	ttl      time.Duration
	now      func() time.Time
}

// NewStore creates a session store where sessions live for ttl.
func NewStore(ttl time.Duration) *Store {
	return &Store{
		sessions: make(map[string]Session),
		ttl:      ttl,
		now:      time.Now,
	}
}

// Create starts a new session and returns its token and CSRF token.
func (s *Store) Create() (token, csrf string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token = randToken()
	csrf = randToken()
	s.sessions[token] = Session{CSRF: csrf, Expiry: s.now().Add(s.ttl)}
	return token, csrf
}

// Valid returns the session for token if it exists and has not expired.
func (s *Store) Valid(token string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[token]
	if !ok {
		return Session{}, false
	}
	if s.now().After(sess.Expiry) {
		delete(s.sessions, token)
		return Session{}, false
	}
	return sess, true
}

// Delete invalidates a session (logout).
func (s *Store) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func randToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("auth: out of randomness: " + err.Error())
	}
	return hex.EncodeToString(b)
}
