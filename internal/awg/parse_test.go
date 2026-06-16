package awg

import (
	"testing"
	"time"
)

// Sample `awg show awg0 dump` output: one interface line + two peer lines.
// Fields are tab-separated. The second peer has never handshaked (0).
const sampleDump = "SRVPRIV=\tSRVPUB=\t51820\toff\n" +
	"PEER1PUB=\tPSK1=\t203.0.113.10:40512\t10.66.66.2/32\t1700000000\t1048576\t524288\t25\n" +
	"PEER2PUB=\t(none)\t(none)\t10.66.66.3/32\t0\t0\t0\toff\n"

func TestParseDump_InterfaceAndPeers(t *testing.T) {
	// Arrange
	now := time.Unix(1700000100, 0) // 100s after peer1's handshake

	// Act
	snap, err := ParseDump("awg0", sampleDump, now)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Interface != "awg0" {
		t.Errorf("interface = %q, want awg0", snap.Interface)
	}
	if snap.ListenPort != 51820 {
		t.Errorf("listen port = %d, want 51820", snap.ListenPort)
	}
	if len(snap.Peers) != 2 {
		t.Fatalf("peers = %d, want 2", len(snap.Peers))
	}

	p1 := snap.Peers[0]
	if p1.PublicKey != "PEER1PUB=" {
		t.Errorf("peer1 key = %q", p1.PublicKey)
	}
	if p1.Endpoint != "203.0.113.10:40512" {
		t.Errorf("peer1 endpoint = %q", p1.Endpoint)
	}
	if p1.RxBytes != 1048576 || p1.TxBytes != 524288 {
		t.Errorf("peer1 transfer = rx %d tx %d", p1.RxBytes, p1.TxBytes)
	}
	if !p1.Online(now) {
		t.Errorf("peer1 should be online (handshake 100s ago)")
	}
}

func TestParseDump_NeverHandshakedIsOffline(t *testing.T) {
	now := time.Unix(1700000100, 0)
	snap, err := ParseDump("awg0", sampleDump, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p2 := snap.Peers[1]
	if !p2.LatestHandshake.IsZero() {
		t.Errorf("peer2 handshake should be zero, got %v", p2.LatestHandshake)
	}
	if p2.Online(now) {
		t.Errorf("peer2 should be offline (never handshaked)")
	}
	// (none)/off placeholders must be normalized to empty strings.
	if p2.Endpoint != "" || p2.Keepalive != "" {
		t.Errorf("peer2 placeholders not cleaned: endpoint=%q keepalive=%q", p2.Endpoint, p2.Keepalive)
	}
}

func TestParseDump_StaleHandshakeIsOffline(t *testing.T) {
	now := time.Unix(1700000000+600, 0) // 10 minutes after handshake
	snap, _ := ParseDump("awg0", sampleDump, now)
	if snap.Peers[0].Online(now) {
		t.Errorf("peer1 should be offline after 10 minutes")
	}
}

func TestSnapshot_Aggregates(t *testing.T) {
	now := time.Unix(1700000100, 0)
	snap, _ := ParseDump("awg0", sampleDump, now)
	if got := snap.OnlineCount(); got != 1 {
		t.Errorf("online count = %d, want 1", got)
	}
	if got := snap.TotalRx(); got != 1048576 {
		t.Errorf("total rx = %d, want 1048576", got)
	}
	if got := snap.TotalTx(); got != 524288 {
		t.Errorf("total tx = %d, want 524288", got)
	}
}

func TestParseDump_Empty(t *testing.T) {
	if _, err := ParseDump("awg0", "   \n  \n", time.Now()); err == nil {
		t.Errorf("expected error for empty dump")
	}
}
