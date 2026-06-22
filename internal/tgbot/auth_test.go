package tgbot

import (
	"path/filepath"
	"testing"

	"github.com/hennessyxo/amneziawg-installer/internal/auth"
)

func TestAuth_AdminAlwaysAllowed(t *testing.T) {
	a := NewAuth([]int64{42}, "", "")
	if !a.IsAuthorized(42) {
		t.Error("preset admin should be authorized")
	}
	if a.IsAuthorized(99) {
		t.Error("non-admin should not be authorized without a password")
	}
}

func TestAuth_Password(t *testing.T) {
	hash, err := auth.HashPassword("Admin2@")
	if err != nil {
		t.Fatal(err)
	}
	a := NewAuth(nil, hash, "")
	if !a.HasPassword() {
		t.Fatal("HasPassword should be true")
	}
	if a.IsAuthorized(7) {
		t.Error("chat should not be authorized before /auth")
	}
	if a.TryPassword(7, "wrong") {
		t.Error("wrong password should fail")
	}
	if a.IsAuthorized(7) {
		t.Error("failed /auth must not authorize")
	}
	if !a.TryPassword(7, "Admin2@") {
		t.Error("correct password should authorize")
	}
	if !a.IsAuthorized(7) {
		t.Error("chat should be authorized after correct /auth")
	}
}

func TestAuth_Persistence(t *testing.T) {
	hash, _ := auth.HashPassword("Admin2@")
	path := filepath.Join(t.TempDir(), "authorized.json")

	a := NewAuth(nil, hash, path)
	if !a.TryPassword(123, "Admin2@") {
		t.Fatal("auth should succeed")
	}

	// A fresh Auth loading the same file should remember the chat.
	b := NewAuth(nil, hash, path)
	if !b.IsAuthorized(123) {
		t.Error("authorized chat should persist across restart")
	}
}

func TestAuth_PasswordDisabled(t *testing.T) {
	a := NewAuth([]int64{1}, "", "")
	if a.HasPassword() {
		t.Error("HasPassword should be false with empty hash")
	}
	if a.TryPassword(5, "anything") {
		t.Error("TryPassword must fail when no password is configured")
	}
}
