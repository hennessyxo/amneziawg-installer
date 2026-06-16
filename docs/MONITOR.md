# awg-monitor

> Live terminal dashboard for a self-hosted **AmneziaWG** VPN — written in Go.
> Живой терминальный монитор для VPN на AmneziaWG.

![go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)
![tui](https://img.shields.io/badge/TUI-Bubble%20Tea-ff69b4)

`awg-monitor` polls `awg show <iface> dump`, resolves client names from the
server config produced by [`amneziawg-install.sh`](../amneziawg-install.sh), and
renders per-client traffic, throughput rates, handshake age, online status, and
inline throughput sparklines — refreshing live in the terminal.

`awg-monitor` опрашивает `awg show <iface> dump`, подтягивает имена клиентов из
конфига сервера и показывает по каждому клиенту: трафик, скорость ↑↓, время
последнего handshake, статус online и спарклайн нагрузки — в реальном времени.

```
  AmneziaWG Monitor   iface awg0   ● online 4/5   ↓ 6.9 GB  ↑ 1.6 GB   15:41:46

   CLIENT         ENDPOINT                  ↓ RATE     ↑ RATE HANDSHAKE  THROUGHPUT
 ● home-pc        198.51.100.6:51820      5.5 MB/s 366.6 KB/s        0s  ▁▂▃▅▆▇▆▅▃▂▁█
 ● laptop         203.0.113.44:51820      8.9 MB/s 730.8 KB/s        0s  ▁▁▂▄▅▇█▇▅▃▂▁
 ● phone-yota     100.64.12.7:41203       2.4 MB/s   2.9 MB/s        0s  ▂▃▄▃▂▁▂▄▆▇▆▄
 ○ old-router     —                          0 B/s      0 B/s     never  ▁▁▁▁▁▁▁▁▁▁▁▁

   refresh 2s · [r] refresh now · [q] quit
```

## Why / Зачем

The most popular AmneziaWG installers are **headless by design** — no metrics, no
dashboard. This fills that gap with a zero-dependency single binary that needs no
Docker, web server, or Grafana stack.

## Install / Установка

The easiest way is the installer menu (option 6), which downloads a prebuilt
binary. To build from source (Go 1.22+) from the repo root:

```bash
go build -o awg-monitor ./cmd/awg-monitor
sudo ./awg-monitor                 # monitors awg0 (needs root for awg)
```

Or cross-compile for your Linux server from any machine:

```bash
GOOS=linux GOARCH=amd64 go build -o awg-monitor ./cmd/awg-monitor   # x86_64
GOOS=linux GOARCH=arm64 go build -o awg-monitor ./cmd/awg-monitor   # ARM (Oracle/RPi)
scp awg-monitor root@server:/usr/local/bin/
```

## Usage / Использование

```bash
awg-monitor                          # monitor awg0
awg-monitor --iface awg0 --interval 1s
awg-monitor --conf /etc/amnezia/amneziawg/awg0.conf   # client names source
awg-monitor --demo                   # synthetic data, no server required
awg-monitor --demo --once            # render one frame to stdout (for screenshots/CI)
```

| Key | Action |
|-----|--------|
| `q` / `Esc` / `Ctrl+C` | quit |
| `r` | refresh now |

## Architecture / Архитектура

```
cmd/awg-monitor/main.go     # flags, awg command source, demo source, --once
internal/
├── awg/                    # dump + config parsing (shared, testable core)
│   ├── parse.go            # ParseDump: `awg show ... dump` → Snapshot
│   └── names.go            # ParseNames: pubkey → client name from server conf
└── ui/                     # Bubble Tea model + pure View(), formatters
    ├── ui.go               # model, rate computation, rendering
    └── format.go           # HumanBytes, HumanRate, Ago, Sparkline
```

`internal/awg` is shared with the web panel (`cmd/awg-panel`).

The parsing layer has no terminal or process dependencies, so it is unit-tested
directly. `View()` is a pure function of model state and is tested without a TTY.

## Tests

```bash
go test ./... -cover
```

Coverage: `internal/awg` ~96%, `internal/ui` ~71%.

## License

MIT — see [../LICENSE](../LICENSE).
