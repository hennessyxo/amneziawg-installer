# awg-monitor

[English](MONITOR.md) · [Русский](MONITOR.ru.md)

> Живой терминальный монитор для VPN на **AmneziaWG**, написан на Go.

![go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)
![tui](https://img.shields.io/badge/TUI-Bubble%20Tea-ff69b4)

`awg-monitor` опрашивает `awg show <iface> dump`, подтягивает имена клиентов из
конфига сервера, созданного [`amneziawg-install.sh`](../amneziawg-install.sh), и
показывает по каждому клиенту трафик, скорость ↑↓, время последнего handshake,
статус online и спарклайн нагрузки, в реальном времени.

```
  AmneziaWG Monitor   iface awg0   ● online 4/5   ↓ 6.9 GB  ↑ 1.6 GB   15:41:46

   CLIENT         ENDPOINT                  ↓ RATE     ↑ RATE HANDSHAKE  THROUGHPUT
 ● home-pc        198.51.100.6:51820      5.5 MB/s 366.6 KB/s        0s  ▁▂▃▅▆▇▆▅▃▂▁█
 ● laptop         203.0.113.44:51820      8.9 MB/s 730.8 KB/s        0s  ▁▁▂▄▅▇█▇▅▃▂▁
 ● phone-yota     100.64.12.7:41203       2.4 MB/s   2.9 MB/s        0s  ▂▃▄▃▂▁▂▄▆▇▆▄
 ○ old-router     —                          0 B/s      0 B/s     never  ▁▁▁▁▁▁▁▁▁▁▁▁

   refresh 2s · [r] refresh now · [q] quit
```

## Зачем

Самые популярные установщики AmneziaWG **изначально без метрик**: ни дашборда, ни
статистики. Этот инструмент закрывает пробел: один бинарник без зависимостей,
без Docker, веб-сервера или стека Grafana.

## Установка

Проще всего через меню установщика (пункт 6): он скачает готовый бинарник. Сборка
из исходников (Go 1.25+) из корня репозитория:

```bash
go build -o awg-monitor ./cmd/awg-monitor
sudo ./awg-monitor                 # мониторит awg0 (нужен root для awg)
```

Или кросс-компиляция под твой Linux-сервер с любой машины:

```bash
GOOS=linux GOARCH=amd64 go build -o awg-monitor ./cmd/awg-monitor   # x86_64
GOOS=linux GOARCH=arm64 go build -o awg-monitor ./cmd/awg-monitor   # ARM (Oracle/RPi)
scp awg-monitor root@server:/usr/local/bin/
```

## Использование

```bash
awg-monitor                          # мониторить awg0
awg-monitor --iface awg0 --interval 1s
awg-monitor --conf /etc/amnezia/amneziawg/awg0.conf   # источник имён клиентов
awg-monitor --demo                   # синтетические данные, сервер не нужен
awg-monitor --demo --once            # один кадр в stdout (для скриншотов/CI)
```

| Клавиша | Действие |
|---------|----------|
| `q` / `Esc` / `Ctrl+C` | выход |
| `r` | обновить сейчас |

## Архитектура

```
cmd/awg-monitor/main.go     # флаги, источник команд awg, демо-источник, --once
internal/
├── awg/                    # разбор dump + конфига (общее тестируемое ядро)
│   ├── parse.go            # ParseDump: `awg show ... dump` → Snapshot
│   └── names.go            # ParseNames: pubkey → имя клиента из конфига сервера
└── ui/                     # модель Bubble Tea + чистая View(), форматтеры
    ├── ui.go               # модель, расчёт скоростей, рендеринг
    └── format.go           # HumanBytes, HumanRate, Ago, Sparkline
```

`internal/awg` общий с веб-панелью (`cmd/awg-panel`).

Слой разбора не зависит от терминала или процессов, поэтому покрыт юнит-тестами
напрямую. `View()` остаётся чистой функцией от состояния модели и тестируется без TTY.

## Тесты

```bash
go test ./... -cover
```

Покрытие: `internal/awg` ~96%, `internal/ui` ~71%.

## Лицензия

MIT, см. [../LICENSE](../LICENSE).
