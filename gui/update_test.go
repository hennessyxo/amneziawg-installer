package main

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.17.0", "v0.16.0", true},
		{"v0.16.0", "v0.16.0", false},
		{"v0.16.1", "v0.16.0", true},
		{"v1.0.0", "v0.99.99", true},
		{"v0.16.0", "v0.17.0", false},
		{"v0.17.0", "dev", false}, // dev build → no nag
		{"garbage", "v0.16.0", false},
	}
	for _, c := range cases {
		if got := isNewer(c.latest, c.current); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestParseVer(t *testing.T) {
	if v, ok := parseVer("v1.2.3"); !ok || v != [3]int{1, 2, 3} {
		t.Errorf("parseVer(v1.2.3) = %v, %v", v, ok)
	}
	if v, ok := parseVer("v0.17.0-rc1"); !ok || v != [3]int{0, 17, 0} {
		t.Errorf("parseVer(rc) = %v, %v", v, ok)
	}
	if _, ok := parseVer("1.2"); ok {
		t.Error("parseVer(1.2) should fail (not 3 parts)")
	}
	if _, ok := parseVer("dev"); ok {
		t.Error("parseVer(dev) should fail")
	}
}
