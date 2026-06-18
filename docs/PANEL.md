# awg-panel

[English](PANEL.md) · [Русский](PANEL.ru.md)

> Web management panel for a self-hosted **AmneziaWG** VPN — Go + htmx, single binary.

![go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)
![ui](https://img.shields.io/badge/UI-htmx-3366cc)

A session-authenticated dashboard for viewing live client traffic and managing
clients in the browser. Built on the same `awg` parsing core as `awg-monitor`.

## Features

- 🔐 **Auth**: admin password (bcrypt), sessions in HttpOnly cookies, CSRF on forms.
- 🔒 **HTTPS**: runs over TLS (a self-signed cert is generated automatically).
- 📊 **Live dashboard**: online status, ↑↓ rates, total traffic per client (htmx polling).
- ➕ **Management**: add / remove / disable / enable / **rename** a client, download `.conf`, QR.
- ✏️ **Edit on the fly**: change an existing client's speed, quota and expiry (the "edit"
  button) — no need to recreate it.
- ⏳ **Quotas & expiry**: when creating a client you set a traffic limit (GB) and/or an
  expiry (days); a background enforcer accounts traffic (reset-aware) and automatically
  **disables** expired and over-quota clients — they are kept and can be **re-enabled**
  (enabling clears a past expiry and resets an exceeded quota).
- 🐢 **Speed limit**: set a cap in Mbit/s — a background `tc` shaper throttles upload
  (an HTB class on the client IP) and download (ingress policing) instead of cutting off.
- 📦 **Single binary**: HTML/CSS/htmx are embedded — nothing to deploy separately.

## Install

Via the installer menu (recommended):

```bash
sudo bash amneziawg-install.sh   # → option 7 "Web panel"
```

The installer downloads the binary, asks for an admin password, generates a
self-signed TLS certificate, and starts a systemd service on `https://<ip>:8443`.

Manually, from source:

```bash
go build -o awg-panel ./cmd/awg-panel
echo 'my-password' | ./awg-panel hash > /etc/amnezia/amneziawg/panel.hash
sudo ./awg-panel \
  --password-hash-file /etc/amnezia/amneziawg/panel.hash \
  --tls-cert cert.pem --tls-key key.pem
```

## Flags

| Flag | Default | Meaning |
|------|---------|---------|
| `--listen` | `:8443` | listen address |
| `--iface` | `awg0` | AmneziaWG interface |
| `--conf` | `/etc/amnezia/amneziawg/awg0.conf` | server config |
| `--params` | `/etc/amnezia/amneziawg/params` | installer parameters |
| `--client-dir` | `/etc/amnezia/amneziawg/clients` | where panel-created configs live |
| `--store` | `/etc/amnezia/amneziawg/clients.json` | lifecycle metadata (quotas/expiry) |
| `--password-hash-file` | `/etc/amnezia/amneziawg/panel.hash` | bcrypt hash of the admin password |
| `--tls-cert` / `--tls-key` | — | enable HTTPS |

`awg-panel hash` reads a password from stdin and prints a bcrypt hash (the plaintext is never stored).

## Security notes

- The panel runs as root (needed for `awg`/`awg-quick`).
- **Brute-force protection:** after 5 failed attempts, logins from that IP are
  locked out for 15 minutes. The password is bcrypt and must satisfy a complexity
  rule (lower- and upper-case, a digit and a special character), enforced by the installer.
- Cookies are `HttpOnly` + `SameSite=Lax`; the `Secure` flag is set under HTTPS.
- Forms carry a session-bound CSRF token.
- **Maximum security (if you're paranoid):** don't open port `8443` to the
  internet — reach the panel over an SSH tunnel instead:
  ```bash
  ssh -L 8443:localhost:8443 root@SERVER   # then open https://localhost:8443
  ```
  The panel is then unreachable from the internet at all, even with the password.
- Clients created during install or via the CLI are **adopted by the panel**
  automatically (you can change limits, disable, rename). Their `.conf` is mirrored
  into the panel directory, so download/QR work for them too.

## Architecture

```
cmd/awg-panel/main.go        # flags, TLS, `hash` subcommand
internal/
├── awgctl/                  # control plane (params, peer add/remove, FileController)
├── auth/                    # bcrypt + in-memory sessions + CSRF
├── lifecycle/               # quota/expiry store, usage accounting, rule engine
├── shaper/                  # tc command planner (per-client bandwidth caps)
├── server/                  # routing, middleware, handlers, rate tracker, enforcer
└── web/                     # embedded templates + static (htmx, CSS)
```

The enforcer (in `server`) reconciles every 30s: it accounts traffic into the
`lifecycle` store, then disables over-quota and expired clients. Bandwidth caps
are re-applied via `shaper` on every change and at startup.

Pure logic (`awgctl`, `auth`) and HTTP handlers (against a fake `Controller`) are
unit-tested; run `go test ./...`.

## License

MIT — see [../LICENSE](../LICENSE).
