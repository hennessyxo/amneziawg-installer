// Package awg parses the output of `awg show <iface> dump` (the AmneziaWG fork
// of WireGuard) into structured snapshots that the UI can render.
package awg

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// onlineWindow defines how recent a handshake must be for a peer to count as
// "online". WireGuard rekeys roughly every ~2 minutes; 3 minutes is a safe
// threshold that avoids flapping.
const onlineWindow = 3 * time.Minute

// Peer is a single connected (or configured) VPN client.
type Peer struct {
	PublicKey       string
	Name            string // resolved from the server config, may be empty
	Endpoint        string
	AllowedIPs      string
	LatestHandshake time.Time // zero value means "never"
	RxBytes         uint64    // bytes received from the peer
	TxBytes         uint64    // bytes sent to the peer
	Keepalive       string
}

// Online reports whether the peer handshook within the online window.
func (p Peer) Online(now time.Time) bool {
	if p.LatestHandshake.IsZero() {
		return false
	}
	return now.Sub(p.LatestHandshake) <= onlineWindow
}

// Snapshot is the state of one interface at a point in time.
type Snapshot struct {
	Interface  string
	ListenPort int
	Peers      []Peer
	Time       time.Time
}

// OnlineCount returns how many peers are currently online.
func (s Snapshot) OnlineCount() int {
	n := 0
	for _, p := range s.Peers {
		if p.Online(s.Time) {
			n++
		}
	}
	return n
}

// TotalRx / TotalTx sum transfer counters across all peers.
func (s Snapshot) TotalRx() uint64 { return sumRx(s.Peers) }
func (s Snapshot) TotalTx() uint64 { return sumTx(s.Peers) }

func sumRx(peers []Peer) uint64 {
	var t uint64
	for _, p := range peers {
		t += p.RxBytes
	}
	return t
}

func sumTx(peers []Peer) uint64 {
	var t uint64
	for _, p := range peers {
		t += p.TxBytes
	}
	return t
}

// ParseDump parses the tab-separated output of `awg show <iface> dump`.
//
// The first non-empty line describes the interface:
//
//	private-key  public-key  listen-port  fwmark
//
// Each following line describes a peer:
//
//	public-key  preshared-key  endpoint  allowed-ips  latest-handshake  rx  tx  keepalive
//
// `now` is stamped onto the returned snapshot so callers can compute rates and
// online status deterministically (and tests can pass a fixed time).
func ParseDump(iface, raw string, now time.Time) (Snapshot, error) {
	snap := Snapshot{Interface: iface, Time: now}

	lines := splitNonEmpty(raw)
	if len(lines) == 0 {
		return snap, fmt.Errorf("awg: empty dump for interface %q", iface)
	}

	// Interface line.
	head := strings.Split(lines[0], "\t")
	if len(head) < 3 {
		return snap, fmt.Errorf("awg: malformed interface line: %q", lines[0])
	}
	if port, err := strconv.Atoi(strings.TrimSpace(head[2])); err == nil {
		snap.ListenPort = port
	}

	// Peer lines.
	for _, line := range lines[1:] {
		f := strings.Split(line, "\t")
		if len(f) < 7 {
			// Not a recognizable peer line; skip defensively.
			continue
		}
		peer := Peer{
			PublicKey:       strings.TrimSpace(f[0]),
			Endpoint:        cleanField(f[2]),
			AllowedIPs:      cleanField(f[3]),
			LatestHandshake: parseHandshake(f[4]),
			RxBytes:         parseUint(f[5]),
			TxBytes:         parseUint(f[6]),
		}
		if len(f) >= 8 {
			peer.Keepalive = cleanField(f[7])
		}
		snap.Peers = append(snap.Peers, peer)
	}
	return snap, nil
}

// splitNonEmpty splits on newlines and drops blank lines.
func splitNonEmpty(raw string) []string {
	var out []string
	for _, l := range strings.Split(raw, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

// cleanField normalizes WireGuard's placeholder values to empty strings.
func cleanField(s string) string {
	s = strings.TrimSpace(s)
	if s == "(none)" || s == "off" {
		return ""
	}
	return s
}

func parseUint(s string) uint64 {
	v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return v
}

// parseHandshake converts a unix-seconds string to a time; 0 means "never".
func parseHandshake(s string) time.Time {
	secs, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || secs == 0 {
		return time.Time{}
	}
	return time.Unix(secs, 0)
}
