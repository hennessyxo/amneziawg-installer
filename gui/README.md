# AmneziaWG Manager (desktop GUI)

A native desktop app — point-and-click front end over the same SSH logic the CLI
(`cmd/awg-deploy`) uses. Connect to a Linux server, install AmneziaWG, and manage
clients without touching a terminal.

Built with [Wails v2](https://wails.io) (Go backend + a tiny vanilla HTML/CSS/JS
frontend — no npm build step). It lives in its **own Go module** (`gui/`) so the
WebKit/GTK-bound Wails dependency never reaches the root module's headless Linux
CI; the shared SSH logic in `internal/deploy` is reused via a `replace` directive.

## What it does

1. **Connect** — server IP, user (default `root`), password or SSH key. Saved
   servers reconnect in one click; the SSH password can be remembered in the OS
   keychain (never written to a file).
2. **Install** — one tuned profile (mobile params: MTU 1280 + Jc=3, reliable on
   both 4G/LTE and PC); optional UDP port under *Advanced*. Streams progress live.
3. **Clients** — list, add (scannable QR + downloadable `.conf`), show any
   client's config/QR later, rename, remove.
4. **Monitoring** — live VPN status, uptime, version and per-client traffic.
5. **Web panel** — install or open the browser panel (it carries per-client
   speed / quota / expiry limits via an always-on server daemon).
6. **Settings** — server info (IP, UDP port, AmneziaWG version, uptime, client
   count), rename the connection, change the web-panel password, or remove the
   panel / AmneziaWG entirely (danger zone).

The UI is bilingual (RU/EN) with a language switch. Scan the QR in the
*AmneziaWG* app, or import the downloaded `.conf` file.

## Develop / build

Requires the [Wails prerequisites](https://wails.io/docs/gettingstarted/installation)
(Go, plus platform WebView: WebKit on macOS, WebView2 on Windows).

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.1

cd gui
wails dev      # live-reload dev window
wails build    # produce build/bin/awg-gui(.app/.exe)
```

CI builds the app on macOS and Windows runners (`.github/workflows/gui.yml`).

## Security notes

- The SSH password is held only in memory for the lifetime of the connection and
  is never written to disk or logged.
- Host keys use **trust-on-first-use**: an unknown server key is recorded in
  `~/.ssh/known_hosts`, but a **changed** key is rejected (possible MITM). The GUI
  has no terminal to confirm on, so first-use trust is automatic — same behaviour
  as the CLI's `--accept-new`.

## Status

The Go backend, frontend wiring and cross-platform build are verified by CI.
Visual/interaction testing is manual (run `wails build` and open the app), since
a GUI cannot be exercised in headless CI.
