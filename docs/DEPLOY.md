# awg-deploy

[English](DEPLOY.md) · [Русский](DEPLOY.ru.md)

> Cross-platform SSH deploy tool — install & manage AmneziaWG on a remote server
> from your own machine (Windows `.exe`, macOS, Linux). One binary, nothing to
> pre-install on the server.

![go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)
![platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-orange)

`awg-deploy` embeds the installer script and pipes it to the server over SSH,
runs it non-interactively, then pulls back the client config, saves it as a
`.conf`, and opens a scannable QR image. No need to SSH in by hand or know any Linux.

## Download

Grab the binary for your OS from [Releases](https://github.com/hennessyxo/amneziawg-installer/releases):

| Your computer | File |
|---------------|------|
| Windows | `awg-deploy-windows-amd64.exe` |
| macOS — Apple Silicon (M1–M5) | `awg-deploy-darwin-arm64.tar.gz` |
| macOS — Intel | `awg-deploy-darwin-amd64.tar.gz` |
| Linux — x86_64 | `awg-deploy-linux-amd64.tar.gz` |
| Linux — ARM | `awg-deploy-linux-arm64.tar.gz` |

> `darwin` = macOS. Download the build for **your own** computer, not the server —
> the server is configured automatically over SSH. The archives preserve the
> executable bit (no `chmod` needed).

### macOS: Gatekeeper

Extract the `.tar.gz` (double-click or `tar -xzf …`). The binary is unsigned, so
clear the quarantine flag once:

```bash
xattr -dr com.apple.quarantine ./awg-deploy
./awg-deploy
```

Or: right-click the file in Finder → **Open** → **Open** (or use
`System Settings → Privacy & Security → Open Anyway`).

## Wizard (easiest)

Run the binary with **no arguments** (on Windows, double-click the `.exe`):

```bash
./awg-deploy-darwin-arm64
```

It asks for the server address and root password, connects over SSH, and runs the
installer and management menu **directly on the server** (over an interactive SSH
session). Everything — install, add/remove/rename clients, monitoring, web panel —
runs server-side; this tool just "logs in" for you. The commands below are for
calling actions directly (scripts/automation).

## Usage

```bash
# Install the VPN on the server (asks for the SSH password if no key is given):
awg-deploy install root@YOUR_SERVER_IP --client phone

# With an SSH key and a non-default SSH port:
awg-deploy install root@YOUR_SERVER_IP:2222 --identity ~/.ssh/id_ed25519

# Add another client (prints its config + QR):
awg-deploy add-client root@YOUR_SERVER_IP laptop

# List clients:
awg-deploy list root@YOUR_SERVER_IP

# Remove a client:
awg-deploy remove-client root@YOUR_SERVER_IP laptop

# Interactive server menu right in your terminal (over SSH):
awg-deploy menu root@YOUR_SERVER_IP

# Live server monitoring from your own terminal:
awg-deploy monitor root@YOUR_SERVER_IP

# Completely remove AmneziaWG from the server (asks to confirm):
awg-deploy uninstall root@YOUR_SERVER_IP
```

> Re-running `install` on a configured server is safe — it detects this and prints
> the management commands (`add-client`, `list`, `remove-client`, `monitor`). The
> interactive menu also lives on the server itself: `sudo bash amneziawg-install.sh`.

On Windows just run the `.exe` from a terminal (PowerShell/Windows Terminal):

```powershell
.\awg-deploy-windows-amd64.exe install root@YOUR_SERVER_IP
```

### install flags

| Flag | Meaning |
|------|---------|
| `--preset` | obfuscation preset; defaults to `mobile` (MTU 1280, Jc=3 — works on mobile and PC) |
| `--port` | AmneziaWG UDP port (default: a free random one) |
| `--client` | first client name |
| `--server-ip` | public IP/host clients connect to (default: autodetect) |
| `--dns1`, `--dns2` | client DNS |
| `--out` | where to save the `.conf` locally |
| `--identity` | SSH private key (otherwise prompts for a password) |
| `--known-hosts` | path to known_hosts |
| `--accept-new` | trust an unknown host without prompting |

## Security

- The host key is verified against `known_hosts`. An unknown host → a trust-on-first-use
  prompt showing the SHA256 fingerprint; a **changed** key → refusal (possible MITM).
- The password is read with hidden input and is never stored.
- The installer script is **embedded in the binary** — nothing extra to download.

## How it works

```
awg-deploy ──SSH──> server
   │  pipes embedded amneziawg-install.sh to `bash -s -- --yes`
   │  passes settings via AWG_* env vars (non-interactive mode)
   │  captures the fenced client config from stdout
   └─ saves <name>.conf + <name>.png (QR) locally and opens the image
```

`monitor` runs `awg show <iface> dump` over SSH on each tick and renders the same
TUI as [`awg-monitor`](MONITOR.md), reusing `internal/awg` and `internal/ui`.

## License

MIT — see [../LICENSE](../LICENSE).
