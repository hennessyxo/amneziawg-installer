package auth

import (
	"testing"
	"time"
)

func TestLimiter_LocksAfterMaxFailures(t *testing.T) {
	l := NewLimiter(3, time.Minute, 10*time.Minute)
	for i := 0; i < 2; i++ {
		l.Fail("ip1")
		if locked, _ := l.Locked("ip1"); locked {
			t.Fatalf("locked too early after %d fails", i+1)
		}
	}
	l.Fail("ip1") // third → locks
	if locked, _ := l.Locked("ip1"); !locked {
		t.Error("should be locked after 3 failures")
	}
	// A different key is unaffected.
	if locked, _ := l.Locked("ip2"); locked {
		t.Error("ip2 should not be locked")
	}
}

func TestLimiter_ResetClears(t *testing.T) {
	l := NewLimiter(2, time.Minute, time.Minute)
	l.Fail("ip")
	l.Fail("ip")
	if locked, _ := l.Locked("ip"); !locked {
		t.Fatal("should be locked")
	}
	l.Reset("ip")
	if locked, _ := l.Locked("ip"); locked {
		t.Error("Reset should clear the lock")
	}
}

func TestLimiter_WindowExpiryResetsCount(t *testing.T) {
	base := time.Unix(1700000000, 0)
	now := base
	l := NewLimiter(3, time.Minute, time.Minute)
	l.now = func() time.Time { return now }

	l.Fail("ip")
	l.Fail("ip")
	now = base.Add(2 * time.Minute) // outside the window → counter restarts
	l.Fail("ip")
	if locked, _ := l.Locked("ip"); locked {
		t.Error("stale failures should not accumulate across the window")
	}
}

func TestLimiter_LockoutExpires(t *testing.T) {
	base := time.Unix(1700000000, 0)
	now := base
	l := NewLimiter(1, time.Minute, 5*time.Minute)
	l.now = func() time.Time { return now }

	l.Fail("ip")
	if locked, _ := l.Locked("ip"); !locked {
		t.Fatal("should lock after 1 failure")
	}
	now = base.Add(6 * time.Minute)
	if locked, _ := l.Locked("ip"); locked {
		t.Error("lockout should have expired")
	}
}
