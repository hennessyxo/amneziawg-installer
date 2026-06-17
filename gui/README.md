# AmneziaWG Manager (desktop GUI)

A native desktop app — point-and-click front end over the same SSH logic the CLI
(`cmd/awg-deploy`) uses. Connect to a Linux server, install AmneziaWG, and manage
clients without touching a terminal.

Built with [Wails v2](https://wails.io) (Go backend + a tiny vanilla HTML/CSS/JS
frontend — no npm build step). It lives in its **own Go module** (`gui/`) so the
WebKit/GTK-bound Wails dependency never reaches the root module's headless Linux
CI; the shared SSH logic in `internal/deploy` is reused via a `replace` directive.

## What it does

1. **Connect** — server IP, user (default `root`), password or SSH key.
2. **Install** — picks the obfuscation preset (обычный / мобильный for 4G/LTE),
   optional UDP port, first client name; streams install progress live.
3. **Manage** — list clients, add a client (shows a scannable QR + downloadable
   `.conf`), remove a client.
4. **Uninstall** — removes AmneziaWG, the panel, all clients and configs.

The QR is read **only** by the standalone *AmneziaWG* app; for AmneziaVPN / other
clients, import the `.conf` file.

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
