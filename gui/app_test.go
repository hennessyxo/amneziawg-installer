package main

import "testing"

func TestValidPanelPassword(t *testing.T) {
	cases := []struct {
		name string
		pw   string
		want bool
	}{
		{"rejects all-digits", "123456", false},
		{"rejects too short even if complex", "Aa1@", false},
		{"accepts the documented example Admin2@", "Admin2@", true},
		{"rejects missing uppercase", "admin2@x", false},
		{"rejects missing lowercase", "ADMIN2@X", false},
		{"rejects missing digit", "Admin@@x", false},
		{"rejects missing special", "Admin234", false},
		{"accepts a long complex password", "Sup3r$ecret!", true},
		{"rejects empty", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := validPanelPassword(c.pw); got != c.want {
				t.Errorf("validPanelPassword(%q) = %v, want %v", c.pw, got, c.want)
			}
		})
	}
}
