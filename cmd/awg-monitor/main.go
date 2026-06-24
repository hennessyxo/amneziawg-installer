// Command awg-monitor is a terminal dashboard for a self-hosted AmneziaWG VPN.
// It polls `awg show <iface> dump`, resolves client names from the server
// config written by amneziawg-install.sh, and renders live per-client traffic,
// handshake age, online status, and throughput sparklines.
//
// Usage:
//
//	awg-monitor                       # monitor awg0 (needs root / awg access)
//	awg-monitor --iface awg0 --interval 1s
//	awg-monitor --demo                # synthetic data, no server required
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/hennessyxo/awg-suite/internal/awg"
	"github.com/hennessyxo/awg-suite/internal/ui"
)

func main() {
	iface := flag.String("iface", "awg0", "AmneziaWG interface name")
	interval := flag.Duration("interval", 2*time.Second, "refresh interval")
	conf := flag.String("conf", "/etc/amnezia/amneziawg/awg0.conf", "server config for client names")
	demo := flag.Bool("demo", false, "run with synthetic data (no server needed)")
	once := flag.Bool("once", false, "render a single frame to stdout and exit (no TTY needed)")
	flag.Parse()

	var src ui.Source
	if *demo {
		src = newDemoSource()
	} else {
		src = cmdSource{iface: *iface, confPath: *conf}
	}

	if *once {
		renderOnce(src, *iface, *interval)
		return
	}

	model := ui.New(src, *iface, *interval)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "awg-monitor:", err)
		os.Exit(1)
	}
}

// renderOnce primes two samples (so throughput rates are populated) and prints
// one rendered frame. Useful for screenshots, docs, and CI smoke tests.
func renderOnce(src ui.Source, iface string, interval time.Duration) {
	model := ui.New(src, iface, interval)
	for i := 0; i < 2; i++ {
		s, err := src.Fetch()
		if err != nil {
			fmt.Fprintln(os.Stderr, "awg-monitor:", err)
			os.Exit(1)
		}
		model.Ingest(s)
		if i == 0 {
			time.Sleep(300 * time.Millisecond)
		}
	}
	fmt.Println(model.View())
}

// cmdSource fetches live data by shelling out to `awg`.
type cmdSource struct {
	iface    string
	confPath string
}

func (c cmdSource) Fetch() (awg.Snapshot, error) {
	out, err := exec.Command("awg", "show", c.iface, "dump").Output()
	if err != nil {
		return awg.Snapshot{}, fmt.Errorf("running `awg show %s dump`: %w (need root and awg installed?)", c.iface, err)
	}
	snap, err := awg.ParseDump(c.iface, string(out), time.Now())
	if err != nil {
		return snap, err
	}
	if data, e := os.ReadFile(c.confPath); e == nil {
		awg.ApplyNames(snap.Peers, awg.ParseNames(string(data)))
	}
	return snap, nil
}

// demoSource generates believable, evolving data so the dashboard can be shown
// (and recorded) without a running VPN server.
type demoSource struct {
	peers []awg.Peer
	rng   *rand.Rand
}

func newDemoSource() *demoSource {
	now := time.Now()
	return &demoSource{
		rng: rand.New(rand.NewSource(now.UnixNano())),
		peers: []awg.Peer{
			{PublicKey: "kPhone1=", Name: "phone-yota", Endpoint: "100.64.12.7:41203", LatestHandshake: now, RxBytes: 824 << 20, TxBytes: 73 << 20},
			{PublicKey: "kLaptop=", Name: "laptop", Endpoint: "203.0.113.44:51820", LatestHandshake: now, RxBytes: 4 << 30, TxBytes: 512 << 20},
			{PublicKey: "kTablet=", Name: "tablet-mts", Endpoint: "10.45.2.9:39114", LatestHandshake: now, RxBytes: 91 << 20, TxBytes: 12 << 20},
			{PublicKey: "kHomePC=", Name: "home-pc", Endpoint: "198.51.100.6:51820", LatestHandshake: now, RxBytes: 2 << 30, TxBytes: 1 << 30},
			{PublicKey: "kBackup=", Name: "old-router", LatestHandshake: time.Time{}},
		},
	}
}

func (d *demoSource) Fetch() (awg.Snapshot, error) {
	now := time.Now()
	for i := range d.peers {
		if d.peers[i].LatestHandshake.IsZero() {
			continue // keep one peer permanently offline
		}
		// Randomly bump counters to simulate live traffic of varying intensity.
		d.peers[i].RxBytes += uint64(d.rng.Intn(3<<20) + 1<<14)
		d.peers[i].TxBytes += uint64(d.rng.Intn(1 << 20))
		// Occasionally let a peer's handshake age so it flips offline.
		if d.rng.Intn(20) == 0 {
			d.peers[i].LatestHandshake = now.Add(-time.Duration(d.rng.Intn(300)) * time.Second)
		} else {
			d.peers[i].LatestHandshake = now
		}
	}
	snap := awg.Snapshot{Interface: "awg0", ListenPort: 51820, Time: now}
	snap.Peers = append(snap.Peers, d.peers...)
	return snap, nil
}
