package awg

import (
	"bufio"
	"strings"
)

// ParseNames extracts a public-key → client-name map from an AmneziaWG server
// config produced by amneziawg-install.sh, which fences each peer like:
//
//	# BEGIN_PEER phone
//	[Peer]
//	PublicKey = <key>
//	...
//	# END_PEER phone
//
// Peers without a BEGIN_PEER marker are simply absent from the map.
func ParseNames(conf string) map[string]string {
	names := make(map[string]string)
	sc := bufio.NewScanner(strings.NewReader(conf))

	var pending string // name from the most recent BEGIN_PEER marker
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "# BEGIN_PEER "):
			pending = strings.TrimSpace(strings.TrimPrefix(line, "# BEGIN_PEER "))
		case strings.HasPrefix(line, "# END_PEER"):
			pending = ""
		case pending != "" && strings.HasPrefix(line, "PublicKey"):
			if key := valueAfterEquals(line); key != "" {
				names[key] = pending
			}
		}
	}
	return names
}

// ApplyNames fills in Peer.Name from the supplied map where a match exists.
func ApplyNames(peers []Peer, names map[string]string) {
	for i := range peers {
		if n, ok := names[peers[i].PublicKey]; ok {
			peers[i].Name = n
		}
	}
}

func valueAfterEquals(line string) string {
	_, val, ok := strings.Cut(line, "=")
	if !ok {
		return ""
	}
	return strings.TrimSpace(val)
}
