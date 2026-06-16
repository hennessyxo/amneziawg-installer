package auth

import (
	"testing"
	"time"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "correct horse battery staple") {
		t.Error("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("wrong password accepted")
	}
}

func TestSessionLifecycle(t *testing.T) {
	s := NewStore(time.Hour)
	token, csrf := s.Create()
	if token == "" || csrf == "" {
		t.Fatal("empty token/csrf")
	}
	sess, ok := s.Valid(token)
	if !ok {
		t.Fatal("freshly created session is invalid")
	}
	if sess.CSRF != csrf {
		t.Error("CSRF mismatch")
	}
	s.Delete(token)
	if _, ok := s.Valid(token); ok {
		t.Error("session still valid after delete")
	}
}

func TestSessionExpiry(t *testing.T) {
	s := NewStore(-time.Second) // sessions are born expired
	token, _ := s.Create()
	if _, ok := s.Valid(token); ok {
		t.Error("expired session reported valid")
	}
}

func TestTokensAreUnique(t *testing.T) {
	s := NewStore(time.Hour)
	t1, _ := s.Create()
	t2, _ := s.Create()
	if t1 == t2 {
		t.Error("session tokens collided")
	}
}
