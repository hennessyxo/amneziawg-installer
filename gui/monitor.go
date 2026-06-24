package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hennessyxo/awg-suite/internal/awg"
	"github.com/hennessyxo/awg-suite/internal/deploy"
	"github.com/hennessyxo/awg-suite/internal/format"
	"github.com/hennessyxo/awg-suite/internal/lifecycle"
)

const awgIface = "awg0"

// lifecycleStore is where the web panel's enforcer keeps per-client usage and the
// daily samples needed to compute traffic over a day/week/month window.
const lifecycleStore = "/etc/amnezia/amneziawg/clients.json"

// HealthResult is the server/VPN health line shown at the top of the manage view.
type HealthResult struct {
	Running bool   `json:"running"` // awg-quick@awg0 active
	Version string `json:"version"` // AmneziaWG tools version
	Uptime  string `json:"uptime"`  // human server uptime
	Clients int    `json:"clients"` // configured peers
}

// TrafficPeer is one client's live transfer state. The human-readable Rx/Tx/
// Handshake are for display; the raw RxBytes/TxBytes/HandshakeUnix let the UI
// sort columns numerically (a formatted "1.2 GB" string can't be compared).
type TrafficPeer struct {
	Name          string `json:"name"`
	Online        bool   `json:"online"`
	Rx            string `json:"rx"`            // human-readable received
	Tx            string `json:"tx"`            // human-readable sent
	Handshake     string `json:"handshake"`     // "2 мин назад" / "—"
	RxBytes       uint64 `json:"rxBytes"`       // raw received, for sorting
	TxBytes       uint64 `json:"txBytes"`       // raw sent, for sorting
	HandshakeUnix int64  `json:"handshakeUnix"` // last handshake unix secs (0 = never)
}

// TrafficResult is the live mini-view of all peers.
type TrafficResult struct {
	Online int           `json:"online"`
	Total  int           `json:"total"`
	Peers  []TrafficPeer `json:"peers"`
}

// ServerHealth reports whether the VPN is up, its version, server uptime and the
// number of configured clients — in a single SSH round-trip.
func (a *App) ServerHealth() (HealthResult, error) {
	cl, t, err := a.conn()
	if err != nil {
		return HealthResult{}, err
	}
	script := "echo \"ACTIVE=$(systemctl is-active awg-quick@" + awgIface + " 2>/dev/null)\"; " +
		"echo \"VER=$(awg --version 2>/dev/null | head -1)\"; " +
		"echo \"UPTIME=$(awk '{print int($1)}' /proc/uptime 2>/dev/null)\"; " +
		"echo \"CLIENTS=$(grep -c '^# BEGIN_PEER' " + serverConf + " 2>/dev/null)\""
	out, err := cl.Run(deploy.Sudo(t.User) + "sh -c " + shellQuote(script))
	if err != nil {
		return HealthResult{}, fmt.Errorf("проверка статуса не удалась: %w", err)
	}
	return parseHealth(out, a.lang), nil
}

// parseHealth turns the KEY=value lines from ServerHealth into a HealthResult.
func parseHealth(out, lang string) HealthResult {
	h := HealthResult{Version: "—"}
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch k {
		case "ACTIVE":
			h.Running = v == "active"
		case "VER":
			if v != "" {
				h.Version = v
			}
		case "UPTIME":
			if secs, err := strconv.Atoi(v); err == nil {
				h.Uptime = formatUptime(secs, lang)
			}
		case "CLIENTS":
			h.Clients, _ = strconv.Atoi(v)
		}
	}
	return h
}

// Traffic returns the live per-client transfer state (poll this from the UI).
func (a *App) Traffic() (TrafficResult, error) {
	cl, t, err := a.conn()
	if err != nil {
		return TrafficResult{}, err
	}
	sudo := deploy.Sudo(t.User)

	names := map[string]string{}
	if confOut, e := cl.Run(deploy.ReadConfCommand(sudo, awgIface)); e == nil {
		names = awg.ParseNames(confOut)
	}
	dump, err := cl.Run(deploy.MonitorDumpCommand(sudo, awgIface))
	if err != nil {
		return TrafficResult{}, fmt.Errorf("не удалось получить статистику: %w", err)
	}
	now := time.Now()
	snap, err := awg.ParseDump(awgIface, dump, now)
	if err != nil {
		return TrafficResult{}, fmt.Errorf("разбор статистики не удался: %w", err)
	}
	awg.ApplyNames(snap.Peers, names)

	res := TrafficResult{Total: len(snap.Peers), Online: snap.OnlineCount()}
	for _, p := range snap.Peers {
		name := p.Name
		if name == "" {
			name = shortKey(p.PublicKey)
		}
		var hsUnix int64
		if !p.LatestHandshake.IsZero() {
			hsUnix = p.LatestHandshake.Unix()
		}
		res.Peers = append(res.Peers, TrafficPeer{
			Name:          name,
			Online:        p.Online(now),
			Rx:            format.HumanBytes(p.RxBytes),
			Tx:            format.HumanBytes(p.TxBytes),
			Handshake:     formatHandshake(p.LatestHandshake, now, a.lang),
			RxBytes:       p.RxBytes,
			TxBytes:       p.TxBytes,
			HandshakeUnix: hsUnix,
		})
	}
	return res, nil
}

// formatUptime renders seconds as a localized "Nd Nh" / "Nh Nm" / "Nm".
func formatUptime(secs int, lang string) string {
	if secs <= 0 {
		return "—"
	}
	d := secs / 86400
	h := (secs % 86400) / 3600
	m := (secs % 3600) / 60
	if lang == "en" {
		switch {
		case d > 0:
			return fmt.Sprintf("%dd %dh", d, h)
		case h > 0:
			return fmt.Sprintf("%dh %dm", h, m)
		default:
			return fmt.Sprintf("%dm", m)
		}
	}
	switch {
	case d > 0:
		return fmt.Sprintf("%d дн %d ч", d, h)
	case h > 0:
		return fmt.Sprintf("%d ч %d мин", h, m)
	default:
		return fmt.Sprintf("%d мин", m)
	}
}

// formatHandshake renders the localized time since the last handshake.
func formatHandshake(t, now time.Time, lang string) string {
	if t.IsZero() {
		return "—"
	}
	d := now.Sub(t)
	if lang == "en" {
		switch {
		case d < time.Minute:
			return "just now"
		case d < time.Hour:
			return fmt.Sprintf("%dm ago", int(d.Minutes()))
		case d < 24*time.Hour:
			return fmt.Sprintf("%dh ago", int(d.Hours()))
		default:
			return fmt.Sprintf("%dd ago", int(d.Hours()/24))
		}
	}
	switch {
	case d < time.Minute:
		return "только что"
	case d < time.Hour:
		return fmt.Sprintf("%d мин назад", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d ч назад", int(d.Hours()))
	default:
		return fmt.Sprintf("%d дн назад", int(d.Hours()/24))
	}
}

// PeriodsResult is the aggregate traffic (summed across all clients) over the
// day/week/month windows. Tracked is false when there is no usage history yet —
// the windows need the web panel's always-on sampler to record daily snapshots.
type PeriodsResult struct {
	Tracked bool   `json:"tracked"`
	Today   string `json:"today"`
	Week    string `json:"week"`
	Month   string `json:"month"`
	AllTime string `json:"allTime"`
}

// TrafficPeriods reads the web panel's lifecycle store from the server and sums
// traffic over today / last 7 days / last 30 days across all clients. Without the
// panel (no store, or no samples recorded yet) it reports Tracked=false so the UI
// can explain that period totals need the panel installed.
func (a *App) TrafficPeriods() (PeriodsResult, error) {
	cl, t, err := a.conn()
	if err != nil {
		return PeriodsResult{}, err
	}
	out, err := cl.Run(deploy.Sudo(t.User) + "cat " + shellQuote(lifecycleStore) + " 2>/dev/null || true")
	if err != nil {
		return PeriodsResult{}, fmt.Errorf("не удалось прочитать историю трафика: %w", err)
	}
	recs := parseLifecycleRecords(out)
	if len(recs) == 0 {
		return PeriodsResult{Tracked: false}, nil
	}
	now := time.Now()
	var today, week, month, all uint64
	tracked := false
	for _, r := range recs {
		all += r.UsedBytes
		if len(r.Samples) > 0 {
			tracked = true
		}
		today += r.Today(now)
		week += r.Last7d(now)
		month += r.Last30d(now)
	}
	if !tracked {
		// Store exists but the sampler hasn't recorded any daily snapshots yet.
		return PeriodsResult{Tracked: false, AllTime: format.HumanBytes(all)}, nil
	}
	return PeriodsResult{
		Tracked: true,
		Today:   format.HumanBytes(today),
		Week:    format.HumanBytes(week),
		Month:   format.HumanBytes(month),
		AllTime: format.HumanBytes(all),
	}, nil
}

// parseLifecycleRecords decodes the panel's clients.json store. A missing/empty
// file yields no records (not an error) so a server without the panel is handled.
func parseLifecycleRecords(out string) []lifecycle.Record {
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	var recs []lifecycle.Record
	if err := json.Unmarshal([]byte(out), &recs); err != nil {
		return nil
	}
	return recs
}

// shortKey is a fallback label for a peer with no resolved name.
func shortKey(pub string) string {
	if len(pub) > 8 {
		return pub[:8] + "…"
	}
	return pub
}
