package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hennessyxo/amneziawg-installer/internal/deploy"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// dial opens an SSH connection using a private key (if identityPath is set) or a
// password, with trust-on-first-use host-key verification.
func dial(t deploy.Target, identityPath, password string) (*deploy.Client, error) {
	auth, err := authMethods(identityPath, password)
	if err != nil {
		return nil, err
	}
	hk, err := tofuHostKey(defaultKnownHosts())
	if err != nil {
		return nil, err
	}
	cl, err := deploy.Dial(t, auth, hk, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться: %w", err)
	}
	return cl, nil
}

func authMethods(identityPath, password string) ([]ssh.AuthMethod, error) {
	if identityPath != "" {
		key, err := os.ReadFile(identityPath)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать ключ %s: %w", identityPath, err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("не удалось разобрать ключ (ключи с паролем не поддерживаются, используйте ssh-agent): %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	if password == "" {
		return nil, errors.New("укажите пароль или путь к SSH-ключу")
	}
	return []ssh.AuthMethod{ssh.Password(password)}, nil
}

// tofuHostKey verifies against known_hosts, trusting an unknown host on first
// use (and recording it) but refusing a CHANGED key — a possible MITM. The GUI
// has no terminal to prompt on, so first-use trust is automatic; this matches
// the CLI's --accept-new behaviour and is documented in the GUI README.
func tofuHostKey(path string) (ssh.HostKeyCallback, error) {
	if err := ensureFile(path); err != nil {
		return nil, err
	}
	base, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить known_hosts: %w", err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := base(hostname, remote, key); err == nil {
			return nil
		} else {
			var ke *knownhosts.KeyError
			if !errors.As(err, &ke) {
				return err
			}
			if len(ke.Want) > 0 {
				return fmt.Errorf("ключ хоста %s ИЗМЕНИЛСЯ — возможна атака MITM, подключение отклонено", hostname)
			}
		}
		return appendKnownHost(path, hostname, key)
	}, nil
}

func appendKnownHost(path, hostname string, key ssh.PublicKey) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)
	_, err = f.WriteString(line + "\n")
	return err
}

func defaultKnownHosts() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "known_hosts"
	}
	return filepath.Join(home, ".ssh", "known_hosts")
}

func ensureFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

func encodeBase64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
