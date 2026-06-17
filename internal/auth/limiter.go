package auth

import (
	"sync"
	"time"
)

// Limiter throttles repeated failed logins per key (typically the client IP) to
// blunt password brute-forcing. After max failures within window, the key is
// locked out for lockout.
type Limiter struct {
	mu      sync.Mutex
	fails   map[string]*attempt
	max     int
	window  time.Duration
	lockout time.Duration
	now     func() time.Time
}

type attempt struct {
	count int
	first time.Time
	until time.Time
}

// NewLimiter builds a Limiter.
func NewLimiter(max int, window, lockout time.Duration) *Limiter {
	return &Limiter{
		fails:   make(map[string]*attempt),
		max:     max,
		window:  window,
		lockout: lockout,
		now:     time.Now,
	}
}

// Locked reports whether key is currently locked out, and until when.
func (l *Limiter) Locked(key string) (bool, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	a := l.fails[key]
	if a == nil {
		return false, time.Time{}
	}
	if l.now().Before(a.until) {
		return true, a.until
	}
	return false, time.Time{}
}

// Fail records a failed attempt; locks the key once it reaches max within window.
func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	a := l.fails[key]
	if a == nil || now.Sub(a.first) > l.window {
		a = &attempt{first: now}
		l.fails[key] = a
	}
	a.count++
	if a.count >= l.max {
		a.until = now.Add(l.lockout)
	}
}

// Reset clears a key's failures (called on a successful login).
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, key)
}
