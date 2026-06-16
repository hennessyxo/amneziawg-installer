package lifecycle

import (
	"path/filepath"
	"testing"
	"time"
)

func tempStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "clients.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return s, path
}

func TestStore_PersistAcrossReopen(t *testing.T) {
	s, path := tempStore(t)
	exp := time.Unix(1800000000, 0)
	if err := s.Put(Record{Name: "phone", PubKey: "PH=", Octet: 2, ExpiresAt: &exp, QuotaBytes: 1 << 30}); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.Get("phone")
	if !ok {
		t.Fatal("record not persisted")
	}
	if got.PubKey != "PH=" || got.Octet != 2 || got.QuotaBytes != 1<<30 {
		t.Errorf("record fields wrong: %+v", got)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(exp) {
		t.Errorf("expiry not persisted: %v", got.ExpiresAt)
	}
}

func TestStore_DeleteAndOctets(t *testing.T) {
	s, _ := tempStore(t)
	_ = s.Put(Record{Name: "a", PubKey: "A=", Octet: 2})
	_ = s.Put(Record{Name: "b", PubKey: "B=", Octet: 5})

	octets := s.UsedOctets()
	if !octets[2] || !octets[5] {
		t.Errorf("UsedOctets = %v, want {2,5}", octets)
	}
	if err := s.Delete("a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get("a"); ok {
		t.Error("record 'a' still present after delete")
	}
	if s.UsedOctets()[2] {
		t.Error("octet 2 should be free after deleting 'a'")
	}
}

func TestApplyUsage_AccumulatesAndHandlesReset(t *testing.T) {
	s, _ := tempStore(t)
	_ = s.Put(Record{Name: "a", PubKey: "A=", Octet: 2})

	// First sample establishes the baseline delta from zero.
	_ = s.ApplyUsage(map[string]Transfer{"A=": {Rx: 1000, Tx: 500}})
	if r, _ := s.Get("a"); r.UsedBytes != 1500 {
		t.Errorf("after first sample UsedBytes = %d, want 1500", r.UsedBytes)
	}

	// Second sample adds the increment.
	_ = s.ApplyUsage(map[string]Transfer{"A=": {Rx: 1000 + 4096, Tx: 500 + 2048}})
	if r, _ := s.Get("a"); r.UsedBytes != 1500+6144 {
		t.Errorf("after second sample UsedBytes = %d, want %d", r.UsedBytes, 1500+6144)
	}

	// Counter reset (interface restart): current < last → counts from zero.
	_ = s.ApplyUsage(map[string]Transfer{"A=": {Rx: 10, Tx: 0}})
	if r, _ := s.Get("a"); r.UsedBytes != 1500+6144+10 {
		t.Errorf("after reset UsedBytes = %d, want %d", r.UsedBytes, 1500+6144+10)
	}
}

func TestEvaluate(t *testing.T) {
	now := time.Unix(1700000000, 0)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name string
		rec  Record
		want Action
	}{
		{"unlimited not expired", Record{}, ActionNone},
		{"not yet expired", Record{ExpiresAt: &future}, ActionNone},
		{"expired is disabled, not deleted", Record{ExpiresAt: &past}, ActionDisable},
		{"under quota", Record{QuotaBytes: 100, UsedBytes: 50}, ActionNone},
		{"over quota", Record{QuotaBytes: 100, UsedBytes: 100}, ActionDisable},
		{"over quota but already disabled", Record{QuotaBytes: 100, UsedBytes: 200, Disabled: true}, ActionNone},
		{"expired but already disabled", Record{ExpiresAt: &past, Disabled: true}, ActionNone},
	}
	for _, c := range cases {
		if got := Evaluate(c.rec, now); got != c.want {
			t.Errorf("%s: Evaluate = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRemainingBytes(t *testing.T) {
	if got := (Record{QuotaBytes: 0}).RemainingBytes(); got != 0 {
		t.Errorf("unlimited RemainingBytes = %d, want 0", got)
	}
	if got := (Record{QuotaBytes: 100, UsedBytes: 30}).RemainingBytes(); got != 70 {
		t.Errorf("RemainingBytes = %d, want 70", got)
	}
	if got := (Record{QuotaBytes: 100, UsedBytes: 150}).RemainingBytes(); got != 0 {
		t.Errorf("exceeded RemainingBytes = %d, want 0", got)
	}
}
