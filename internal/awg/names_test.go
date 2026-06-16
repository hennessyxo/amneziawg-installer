package awg

import "testing"

const sampleConf = `[Interface]
Address = 10.66.66.1/24
ListenPort = 51820
PrivateKey = SRVPRIV=

# BEGIN_PEER phone
[Peer]
PublicKey = PEER1PUB=
PresharedKey = PSK1=
AllowedIPs = 10.66.66.2/32
# END_PEER phone

# BEGIN_PEER laptop
[Peer]
PublicKey = PEER2PUB=
AllowedIPs = 10.66.66.3/32
# END_PEER laptop
`

func TestParseNames(t *testing.T) {
	names := ParseNames(sampleConf)
	if got := names["PEER1PUB="]; got != "phone" {
		t.Errorf("PEER1 name = %q, want phone", got)
	}
	if got := names["PEER2PUB="]; got != "laptop" {
		t.Errorf("PEER2 name = %q, want laptop", got)
	}
	if len(names) != 2 {
		t.Errorf("names size = %d, want 2", len(names))
	}
}

func TestApplyNames(t *testing.T) {
	peers := []Peer{{PublicKey: "PEER1PUB="}, {PublicKey: "UNKNOWN="}}
	ApplyNames(peers, ParseNames(sampleConf))
	if peers[0].Name != "phone" {
		t.Errorf("peer0 name = %q, want phone", peers[0].Name)
	}
	if peers[1].Name != "" {
		t.Errorf("unknown peer should keep empty name, got %q", peers[1].Name)
	}
}
