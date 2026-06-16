package awgctl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ipBase is the /24 the installer uses for the VPN subnet (server is .1).
const ipBase = "10.66.66."

var (
	nameRe    = regexp.MustCompile(`(?m)^# BEGIN_PEER (.+)$`)
	octetRe   = regexp.MustCompile(`10\.66\.66\.(\d+)/32`)
	validName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)
)

// SanitizeName trims a client name to the safe character set used by the
// installer. It returns the cleaned name and whether it is non-empty/valid.
func SanitizeName(raw string) (string, bool) {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
		if b.Len() >= 32 {
			break
		}
	}
	name := b.String()
	return name, name != "" && validName.MatchString(name)
}

// ListPeerNames returns the client names fenced in the server config.
func ListPeerNames(conf string) []string {
	var names []string
	for _, m := range nameRe.FindAllStringSubmatch(conf, -1) {
		names = append(names, strings.TrimSpace(m[1]))
	}
	return names
}

// HasPeer reports whether a peer with the given name already exists.
func HasPeer(conf, name string) bool {
	for _, n := range ListPeerNames(conf) {
		if n == name {
			return true
		}
	}
	return false
}

// NextOctet returns the lowest free host octet in 10.66.66.0/24 (2..254).
func NextOctet(conf string) (int, error) {
	used := map[int]bool{1: true} // .1 is the server
	for _, m := range octetRe.FindAllStringSubmatch(conf, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil {
			used[n] = true
		}
	}
	for i := 2; i <= 254; i++ {
		if !used[i] {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no free addresses in %s0/24", ipBase)
}

// PeerBlock renders a fenced [Peer] block in the exact format the installer
// uses, so the CLI menu and the web panel stay interoperable.
func PeerBlock(name, pubKey, psk string, octet int) string {
	return fmt.Sprintf(`
# BEGIN_PEER %s
[Peer]
PublicKey = %s
PresharedKey = %s
AllowedIPs = %s%d/32,fd42:42:42::%d/128
# END_PEER %s
`, name, pubKey, psk, ipBase, octet, octet, name)
}

// AddPeerBlock appends a peer block to the server config.
func AddPeerBlock(conf, name, pubKey, psk string, octet int) string {
	return strings.TrimRight(conf, "\n") + "\n" + PeerBlock(name, pubKey, psk, octet)
}

// AppendBlock appends a pre-rendered peer block (used to re-enable a client).
func AppendBlock(conf, block string) string {
	return strings.TrimRight(conf, "\n") + "\n" + block
}

// FreeOctet returns the lowest free host octet, avoiding both the octets present
// in the live config and any reserved by lifecycle records (e.g. disabled
// clients that may be re-enabled).
func FreeOctet(conf string, reserved map[int]bool) (int, error) {
	used := map[int]bool{1: true} // .1 is the server
	for _, m := range octetRe.FindAllStringSubmatch(conf, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil {
			used[n] = true
		}
	}
	for o := range reserved {
		used[o] = true
	}
	for i := 2; i <= 254; i++ {
		if !used[i] {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no free addresses in %s0/24", ipBase)
}

// RemovePeerBlock deletes the fenced block for name. It returns the new config
// and whether a block was actually removed.
func RemovePeerBlock(conf, name string) (string, bool) {
	begin := "# BEGIN_PEER " + name
	end := "# END_PEER " + name
	lines := strings.Split(conf, "\n")
	out := make([]string, 0, len(lines))
	removed, inBlock := false, false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == begin {
			inBlock, removed = true, true
			continue
		}
		if inBlock {
			if t == end {
				inBlock = false
			}
			continue
		}
		out = append(out, l)
	}
	cleaned := collapseBlankLines(strings.Join(out, "\n"))
	return cleaned, removed
}

// RenderClientConfig builds a client .conf. Obfuscation parameters MUST match
// the server, so they are copied verbatim from Params.
func RenderClientConfig(p Params, privKey, psk string, octet int) string {
	return fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s%d/32,fd42:42:42::%d/128
DNS = %s,%s
MTU = %s
Jc = %s
Jmin = %s
Jmax = %s
S1 = %s
S2 = %s
H1 = %s
H2 = %s
H3 = %s
H4 = %s

[Peer]
PublicKey = %s
PresharedKey = %s
Endpoint = %s:%s
AllowedIPs = 0.0.0.0/0,::/0
PersistentKeepalive = 25
`,
		privKey, ipBase, octet, octet,
		p.ClientDNS1, p.ClientDNS2, p.ClientMTU,
		p.Jc, p.Jmin, p.Jmax, p.S1, p.S2, p.H1, p.H2, p.H3, p.H4,
		p.ServerPubKey, psk, p.ServerPubIP, p.ServerPort,
	)
}

// collapseBlankLines squeezes runs of blank lines into a single one.
func collapseBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	prevBlank := false
	for _, l := range lines {
		blank := strings.TrimSpace(l) == ""
		if blank && prevBlank {
			continue
		}
		out = append(out, l)
		prevBlank = blank
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}
