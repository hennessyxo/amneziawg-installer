package tgbot

import (
	"path/filepath"
	"testing"

	"github.com/hennessyxo/amneziawg-installer/internal/auth"
)

func TestAuth_RequiresBothListAndPassword(t *testing.T) {
	hash, err := auth.HashPassword("Admin2@")
	if err != nil {
		t.Fatal(err)
	}
	a := NewAuth([]int64{42}, hash, "")

	// Admin who has not entered the password is NOT authorized yet.
	if a.IsAuthorized(42) {
		t.Error("admin must not be authorized before /auth")
	}
	if !a.IsAdmin(42) {
		t.Error("42 should be recognized as an admin")
	}

	// Admin + correct password → authorized.
	if !a.TryPassword(42, "Admin2@") {
		t.Fatal("admin with correct password should authenticate")
	}
	if !a.IsAuthorized(42) {
		t.Error("admin who passed the password should be authorized")
	}
}

func TestAuth_NonAdminNeverAllowed(t *testing.T) {
	hash, _ := auth.HashPassword("Admin2@")
	a := NewAuth([]int64{42}, hash, "")

	// A non-admin cannot authenticate even with the correct password.
	if a.TryPassword(99, "Admin2@") {
		t.Error("non-admin must not authenticate, even with the right password")
	}
	if a.IsAuthorized(99) {
		t.Error("non-admin must never be authorized")
	}
}

func TestAuth_WrongPassword(t *testing.T) {
	hash, _ := auth.HashPassword("Admin2@")
	a := NewAuth([]int64{7}, hash, "")
	if a.TryPassword(7, "nope") {
		t.Error("wrong password should fail")
	}
	if a.IsAuthorized(7) {
		t.Error("failed /auth must not authorize")
	}
}

func TestAuth_Persistence(t *testing.T) {
	hash, _ := auth.HashPassword("Admin2@")
	path := filepath.Join(t.TempDir(), "authorized.json")

	a := NewAuth([]int64{123}, hash, path)
	if !a.TryPassword(123, "Admin2@") {
		t.Fatal("auth should succeed")
	}

	// A fresh Auth loading the same file should keep the admin authorized.
	b := NewAuth([]int64{123}, hash, path)
	if !b.IsAuthorized(123) {
		t.Error("authorized admin should persist across restart")
	}

	// But if that ID is no longer an admin, persisted password does not help.
	c := NewAuth(nil, hash, path)
	if c.IsAuthorized(123) {
		t.Error("removed-from-allowlist user must lose access despite persisted password")
	}
}
