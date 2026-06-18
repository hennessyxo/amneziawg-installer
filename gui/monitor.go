package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hennessyxo/amneziawg-installer/internal/awg"
	"github.com/hennessyxo/amneziawg-installer/internal/deploy"
	"github.com/hennessyxo/amneziawg-installer/internal/format"
)

const awgIface = "awg0"

// HealthResult is the server/VPN health line shown at the top of the manage view.
type HealthResult struct {
	Running bool   `json:"running"` // awg-quick@awg0 active
	Version string `json:"version"` // AmneziaWG tools version
	Uptime  string `json:"uptime"`  // human server uptime
	Clients int    `json:"clients"` // configured peers
}

// TrafficPeer is one client's live transfer state.
type TrafficPeer struct {
	Name      string `json:"name"`
	Online    bool   `json:"online"`
	Rx        string `json:"rx"`        // human-readable received
	Tx        string `json:"tx"`        // human-readable sent
	Handshake string `json:"handshake"` // "2 мин назад" / "—"
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
		res.Peers = append(res.Peers, TrafficPeer{
			Name:      name,
			Online:    p.Online(now),
			Rx:        format.HumanBytes(p.RxBytes),
			Tx:        format.HumanBytes(p.TxBytes),
			Handshake: formatHandshake(p.LatestHandshake, now, a.lang),
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

// shortKey is a fallback label for a peer with no resolved name.
func shortKey(pub string) string {
	if len(pub) > 8 {
		return pub[:8] + "…"
	}
	return pub
}
