package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/hennessyxo/awg-suite/internal/awg"
)

func TestView_RendersPeers(t *testing.T) {
	now := time.Unix(1700000100, 0)
	snap := awg.Snapshot{
		Interface:  "awg0",
		ListenPort: 51820,
		Time:       now,
		Peers: []awg.Peer{
			{PublicKey: "AAA=", Name: "phone", Endpoint: "203.0.113.5:40512",
				LatestHandshake: now.Add(-20 * time.Second), RxBytes: 1048576, TxBytes: 2048},
			{PublicKey: "BBB=", Name: "laptop", LatestHandshake: time.Time{}},
		},
	}

	m := New(nil, "awg0", 2*time.Second)
	m.ingest(snap)
	out := m.View()

	for _, want := range []string{"AmneziaWG Monitor", "awg0", "phone", "laptop", "online 1/2"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q\n---\n%s", want, out)
		}
	}
}

func TestIngest_ComputesRates(t *testing.T) {
	t0 := time.Unix(1700000000, 0)
	t1 := t0.Add(2 * time.Second)

	m := New(nil, "awg0", time.Second)
	m.ingest(awg.Snapshot{Time: t0, Peers: []awg.Peer{{PublicKey: "AAA=", RxBytes: 1000, TxBytes: 500}}})
	m.ingest(awg.Snapshot{Time: t1, Peers: []awg.Peer{{PublicKey: "AAA=", RxBytes: 1000 + 4096, TxBytes: 500 + 2048}}})

	if got := m.rx["AAA="]; got != 2048 { // 4096 bytes / 2s
		t.Errorf("rx rate = %.0f, want 2048", got)
	}
	if got := m.tx["AAA="]; got != 1024 { // 2048 bytes / 2s
		t.Errorf("tx rate = %.0f, want 1024", got)
	}
	if len(m.hist["AAA="]) != 1 {
		t.Errorf("history length = %d, want 1", len(m.hist["AAA="]))
	}
}

func TestIngest_CounterResetYieldsZero(t *testing.T) {
	t0 := time.Unix(1700000000, 0)
	m := New(nil, "awg0", time.Second)
	m.ingest(awg.Snapshot{Time: t0, Peers: []awg.Peer{{PublicKey: "AAA=", RxBytes: 9999}}})
	m.ingest(awg.Snapshot{Time: t0.Add(time.Second), Peers: []awg.Peer{{PublicKey: "AAA=", RxBytes: 10}}})
	if got := m.rx["AAA="]; got != 0 {
		t.Errorf("rate after counter reset = %.0f, want 0", got)
	}
}
