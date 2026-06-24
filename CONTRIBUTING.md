# Contributing to AWG Suite

Thanks for taking the time to contribute. This project is a toolkit for
self-hosting an [AmneziaWG](https://github.com/amnezia-vpn) VPN: a bash installer,
a few Go tools, and a desktop app. Bug reports, fixes, and focused features are all
welcome.

## Project layout

```
amneziawg-install.sh     # bash installer + on-server management menu
cmd/                     # Go entrypoints
  awg-monitor/           # terminal traffic dashboard (Bubble Tea TUI)
  awg-panel/             # web management panel (Go + htmx)
  awg-bot/               # Telegram bot
  awg-deploy/            # cross-platform SSH deploy tool
internal/                # shared packages (awg, awgctl, auth, lifecycle, ...)
gui/                     # desktop app (Wails v2, its own Go module)
docs/                    # per-tool docs, bilingual (X.md = EN, X.ru.md = RU)
```

The root is a single Go module. The GUI lives in its own module under `gui/` so its
WebKit/GTK-bound Wails dependency never reaches the root module's Linux CI.

## Prerequisites

- Go 1.25 or newer (the build uses `go.mod`'s version).
- `shellcheck` for the installer script.
- For the desktop app: the [Wails v2 prerequisites](https://wails.io/docs/gettingstarted/installation)
  and the Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.1`).

## Build and test

Root module:

```bash
go build ./...
go test ./... -race -cover
gofmt -l .                       # should print nothing
```

Installer script:

```bash
bash -n amneziawg-install.sh
shellcheck amneziawg-install.sh
```

Desktop app:

```bash
cd gui
go build ./...
go test ./...
wails build                      # optional, produces build/bin/awg-gui
```

CI runs the equivalent checks on every pull request, so green CI is the bar for
merging. The GUI cannot be exercised in headless CI, so visual changes are verified
by building and opening the app locally.

## Conventions

- Format Go with `gofmt`/`goimports`. Keep functions small and packages cohesive.
- User-facing strings and docs are bilingual (Russian and English). If you add or
  change UI text, the installer prompts, the panel, the GUI, or a doc, update both
  languages.
- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org):
  `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, `perf:`, `ci:`.
- Keep each pull request focused on one thing. Smaller diffs get reviewed faster.

## Testing the live VPN path

Most logic (parsing, auth, lifecycle, command planning) is unit-tested and runs in
CI. The parts that touch a real server (installing AmneziaWG, the SSH paths, the `tc`
shaper) need a live Ubuntu/Debian VPS to verify end to end. If your change affects
those paths, please say in the pull request how you tested it, or note that it still
needs a live run.

## Reporting bugs and proposing features

Open an issue using one of the templates. For anything security-related, do not open
a public issue: see [SECURITY.md](SECURITY.md).

By contributing, you agree that your contributions are licensed under the project's
[MIT License](LICENSE).
