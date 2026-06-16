// Package ui implements the terminal dashboard for AmneziaWG using Bubble Tea.
package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hennessyxo/amneziawg-installer/internal/awg"
)

const (
	maxHistory  = 60
	sparkWidth  = 12
	nameWidth   = 14
	endpntWidth = 21
)

// Color palette (kept intentional, not default terminal colors).
var (
	accent   = lipgloss.Color("#37d4a0") // AmneziaWG green
	dim      = lipgloss.Color("#6b7280")
	warnCol  = lipgloss.Color("#f59e0b")
	offCol   = lipgloss.Color("#4b5563")
	titleBg  = lipgloss.Color("#0f3d2e")
	titleStl = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#eafff5")).Background(titleBg).Padding(0, 1)
	hdrStl   = lipgloss.NewStyle().Foreground(dim).Bold(true)
	dimStl   = lipgloss.NewStyle().Foreground(dim)
	accStl   = lipgloss.NewStyle().Foreground(accent)
	onDot    = lipgloss.NewStyle().Foreground(accent).Render("●")
	offDot   = lipgloss.NewStyle().Foreground(offCol).Render("○")
)

type tickMsg time.Time
type snapMsg struct {
	snap awg.Snapshot
	err  error
}

// Model is the Bubble Tea model backing the dashboard.
type Model struct {
	src      Source
	iface    string
	interval time.Duration
	now      func() time.Time // injectable clock for tests

	cur      awg.Snapshot
	hasPrev  bool
	rx, tx   map[string]float64   // pubkey -> current rate (bytes/s)
	hist     map[string][]float64 // pubkey -> recent total throughput
	err      error
	w, h     int
	quitting bool
}

// New builds a Model. The clock defaults to time.Now.
func New(src Source, iface string, interval time.Duration) Model {
	return Model{
		src:      src,
		iface:    iface,
		interval: interval,
		now:      time.Now,
		rx:       map[string]float64{},
		tx:       map[string]float64{},
		hist:     map[string][]float64{},
		w:        100,
	}
}

func (m Model) Init() tea.Cmd { return m.fetch() }

// Ingest applies a snapshot outside the Bubble Tea event loop. It is used by
// the non-interactive `--once` render path (and by tests).
func (m *Model) Ingest(s awg.Snapshot) { m.ingest(s) }

func (m Model) fetch() tea.Cmd {
	return func() tea.Msg {
		s, err := m.src.Fetch()
		return snapMsg{snap: s, err: err}
	}
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Update handles messages and drives the refresh loop.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "r":
			return m, m.fetch()
		}

	case snapMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tick(m.interval)
		}
		m.ingest(msg.snap)
		return m, tick(m.interval)

	case tickMsg:
		return m, m.fetch()
	}
	return m, nil
}

// ingest stores a fresh snapshot and recomputes per-peer rates and history.
func (m *Model) ingest(s awg.Snapshot) {
	m.err = nil
	// Rates are computed by comparing the incoming snapshot against the most
	// recently stored one (m.cur).
	if m.hasPrev {
		dt := s.Time.Sub(m.cur.Time).Seconds()
		if dt > 0 {
			prevByKey := indexByKey(m.cur.Peers)
			for _, p := range s.Peers {
				old, ok := prevByKey[p.PublicKey]
				if !ok {
					continue
				}
				rx := deltaRate(p.RxBytes, old.RxBytes, dt)
				tx := deltaRate(p.TxBytes, old.TxBytes, dt)
				m.rx[p.PublicKey] = rx
				m.tx[p.PublicKey] = tx
				m.hist[p.PublicKey] = appendCapped(m.hist[p.PublicKey], rx+tx, maxHistory)
			}
		}
	}
	m.cur = s
	m.hasPrev = true
}

// View renders the dashboard. It is a pure function of model state, so it can
// be unit-tested without a real terminal.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	var b strings.Builder
	now := m.cur.Time
	if now.IsZero() {
		now = m.now()
	}

	online := m.cur.OnlineCount()
	title := titleStl.Render("AmneziaWG Monitor")
	stats := fmt.Sprintf(" %s  iface %s   %s online %d/%d   ↓ %s  ↑ %s   %s",
		title,
		accStl.Render(m.iface),
		onDot, online, len(m.cur.Peers),
		HumanBytes(m.cur.TotalRx()), HumanBytes(m.cur.TotalTx()),
		dimStl.Render(now.Format("15:04:05")),
	)
	b.WriteString(stats)
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(warnCol).Render("⚠ "+m.err.Error()) + "\n\n")
	}

	header := fmt.Sprintf("   %-*s %-*s %10s %10s %9s  %s",
		nameWidth, "CLIENT", endpntWidth, "ENDPOINT", "↓ RATE", "↑ RATE", "HANDSHAKE", "THROUGHPUT")
	b.WriteString(hdrStl.Render(header) + "\n")

	if len(m.cur.Peers) == 0 {
		b.WriteString(dimStl.Render("   no peers configured") + "\n")
	}

	for _, p := range sortedPeers(m.cur.Peers, now) {
		b.WriteString(m.row(p, now) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStl.Render(fmt.Sprintf("   refresh %s · [r] refresh now · [q] quit", m.interval)))
	return b.String()
}

func (m Model) row(p awg.Peer, now time.Time) string {
	dot := offDot
	if p.Online(now) {
		dot = onDot
	}
	name := p.Name
	if name == "" {
		name = shortKey(p.PublicKey)
	}
	endpoint := p.Endpoint
	if endpoint == "" {
		endpoint = "—"
	}
	spark := accStl.Render(Sparkline(m.hist[p.PublicKey], sparkWidth))

	return fmt.Sprintf(" %s %-*s %-*s %10s %10s %9s  %s",
		dot,
		nameWidth, truncate(name, nameWidth),
		endpntWidth, truncate(endpoint, endpntWidth),
		HumanRate(m.rx[p.PublicKey]),
		HumanRate(m.tx[p.PublicKey]),
		Ago(p.LatestHandshake, now),
		spark,
	)
}

// --- helpers ---------------------------------------------------------------

func indexByKey(peers []awg.Peer) map[string]awg.Peer {
	m := make(map[string]awg.Peer, len(peers))
	for _, p := range peers {
		m[p.PublicKey] = p
	}
	return m
}

func deltaRate(cur, old uint64, dt float64) float64 {
	if cur < old || dt <= 0 { // counter reset or no time elapsed
		return 0
	}
	return float64(cur-old) / dt
}

func appendCapped(s []float64, v float64, max int) []float64 {
	s = append(s, v)
	if len(s) > max {
		s = s[len(s)-max:]
	}
	return s
}

// sortedPeers lists online peers first, then alphabetically by display name.
func sortedPeers(peers []awg.Peer, now time.Time) []awg.Peer {
	out := make([]awg.Peer, len(peers))
	copy(out, peers)
	sort.SliceStable(out, func(i, j int) bool {
		oi, oj := out[i].Online(now), out[j].Online(now)
		if oi != oj {
			return oi
		}
		return displayName(out[i]) < displayName(out[j])
	})
	return out
}

func displayName(p awg.Peer) string {
	if p.Name != "" {
		return p.Name
	}
	return p.PublicKey
}

func shortKey(k string) string {
	k = strings.TrimSuffix(k, "=")
	if len(k) > 8 {
		return k[:8]
	}
	return k
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
