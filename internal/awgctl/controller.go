package awgctl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hennessyxo/amneziawg-installer/internal/awg"
)

// Client is a generated VPN client and its full configuration text.
type Client struct {
	Name   string
	Config string
}

// Controller is the panel's view of the running AmneziaWG server. It is an
// interface so HTTP handlers can be tested against a fake.
type Controller interface {
	Snapshot() (awg.Snapshot, error)
	ListClients() ([]string, error)
	AddClient(name string) (Client, error)
	RevokeClient(name string) error
	ClientConfig(name string) (string, error)
}

// FileController is the production Controller: it shells out to `awg` and edits
// the server config on disk. It is intentionally thin — all the parsing and
// config-rewriting logic lives in the unit-tested pure functions above.
type FileController struct {
	Iface     string // e.g. "awg0"
	ConfPath  string // /etc/amnezia/amneziawg/awg0.conf
	ParamPath string // /etc/amnezia/amneziawg/params
	ClientDir string // where panel-generated client .conf files are stored
}

func (c FileController) Snapshot() (awg.Snapshot, error) {
	out, err := exec.Command("awg", "show", c.Iface, "dump").Output()
	if err != nil {
		return awg.Snapshot{}, fmt.Errorf("awg show: %w", err)
	}
	snap, err := awg.ParseDump(c.Iface, string(out), time.Now())
	if err != nil {
		return snap, err
	}
	if data, e := os.ReadFile(c.ConfPath); e == nil {
		awg.ApplyNames(snap.Peers, awg.ParseNames(string(data)))
	}
	return snap, nil
}

func (c FileController) ListClients() ([]string, error) {
	data, err := os.ReadFile(c.ConfPath)
	if err != nil {
		return nil, err
	}
	return ListPeerNames(string(data)), nil
}

func (c FileController) AddClient(name string) (Client, error) {
	confBytes, err := os.ReadFile(c.ConfPath)
	if err != nil {
		return Client{}, err
	}
	conf := string(confBytes)
	if HasPeer(conf, name) {
		return Client{}, fmt.Errorf("client %q already exists", name)
	}

	paramBytes, err := os.ReadFile(c.ParamPath)
	if err != nil {
		return Client{}, err
	}
	params := ParseParams(string(paramBytes))

	priv, err := c.genKey()
	if err != nil {
		return Client{}, err
	}
	pub, err := c.pubKey(priv)
	if err != nil {
		return Client{}, err
	}
	psk, err := c.genPSK()
	if err != nil {
		return Client{}, err
	}
	octet, err := NextOctet(conf)
	if err != nil {
		return Client{}, err
	}

	clientCfg := RenderClientConfig(params, priv, psk, octet)
	newConf := AddPeerBlock(conf, name, pub, psk, octet)

	if err := os.WriteFile(c.ConfPath, []byte(newConf), 0o600); err != nil {
		return Client{}, err
	}
	if err := os.MkdirAll(c.ClientDir, 0o700); err != nil {
		return Client{}, err
	}
	if err := os.WriteFile(c.clientFile(name), []byte(clientCfg), 0o600); err != nil {
		return Client{}, err
	}
	if err := c.syncConf(); err != nil {
		return Client{}, err
	}
	return Client{Name: name, Config: clientCfg}, nil
}

func (c FileController) RevokeClient(name string) error {
	confBytes, err := os.ReadFile(c.ConfPath)
	if err != nil {
		return err
	}
	newConf, removed := RemovePeerBlock(string(confBytes), name)
	if !removed {
		return fmt.Errorf("client %q not found", name)
	}
	if err := os.WriteFile(c.ConfPath, []byte(newConf), 0o600); err != nil {
		return err
	}
	_ = os.Remove(c.clientFile(name))
	return c.syncConf()
}

func (c FileController) ClientConfig(name string) (string, error) {
	data, err := os.ReadFile(c.clientFile(name))
	if err != nil {
		return "", fmt.Errorf("config for %q unavailable (only panel-created clients are stored): %w", name, err)
	}
	return string(data), nil
}

// --- exec helpers ----------------------------------------------------------

func (c FileController) clientFile(name string) string {
	return filepath.Join(c.ClientDir, c.Iface+"-client-"+name+".conf")
}

func (c FileController) genKey() (string, error) { return runOut("awg", "genkey") }
func (c FileController) genPSK() (string, error) { return runOut("awg", "genpsk") }

func (c FileController) pubKey(priv string) (string, error) {
	cmd := exec.Command("awg", "pubkey")
	cmd.Stdin = strings.NewReader(priv + "\n")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("awg pubkey: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// syncConf applies config changes to the live interface without dropping peers.
func (c FileController) syncConf() error {
	stripped, err := exec.Command("awg-quick", "strip", c.Iface).Output()
	if err != nil {
		return fmt.Errorf("awg-quick strip: %w", err)
	}
	tmp, err := os.CreateTemp("", "awg-sync-*.conf")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(stripped); err != nil {
		return err
	}
	tmp.Close()
	if err := exec.Command("awg", "syncconf", c.Iface, tmp.Name()).Run(); err != nil {
		return fmt.Errorf("awg syncconf: %w", err)
	}
	return nil
}

func runOut(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}
