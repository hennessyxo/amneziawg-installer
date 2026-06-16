# awg-panel

> Web management panel for a self-hosted **AmneziaWG** VPN — Go + htmx, single binary.
> Веб-панель управления VPN на AmneziaWG.

![go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)
![ui](https://img.shields.io/badge/UI-htmx-3366cc)

A session-authenticated dashboard for viewing live client traffic and managing
clients in the browser. Built on the same `awg` parsing core as `awg-monitor`.

## Features

- 🔐 **Авторизация**: пароль администратора (bcrypt), сессии в HttpOnly-куках, CSRF на формах
- 🔒 **HTTPS**: работает по TLS (самоподписанный серт ставится автоматически)
- 📊 **Живой дашборд**: онлайн-статус, скорость ↑↓, суммарный трафик по клиентам (htmx-поллинг)
- ➕ **Управление**: добавить / удалить клиента, скачать `.conf`, QR-код — без перезапуска VPN
- 📦 **Один бинарник**: HTML/CSS/htmx вшиты через `embed` — нечего деплоить отдельно

## Install / Установка

Через меню установщика (рекомендуется):

```bash
sudo bash amneziawg-install.sh   # → пункт 7 «Веб-панель»
```

Установщик скачает бинарник, спросит пароль администратора, сгенерирует
самоподписанный TLS-сертификат и поднимет systemd-службу на `https://<ip>:8443`.

Вручную из исходников:

```bash
go build -o awg-panel ./cmd/awg-panel
echo 'мой-пароль' | ./awg-panel hash > /etc/amnezia/amneziawg/panel.hash
sudo ./awg-panel \
  --password-hash-file /etc/amnezia/amneziawg/panel.hash \
  --tls-cert cert.pem --tls-key key.pem
```

## Flags

| Flag | Default | Назначение |
|------|---------|-----------|
| `--listen` | `:8443` | адрес прослушивания |
| `--iface` | `awg0` | интерфейс AmneziaWG |
| `--conf` | `/etc/amnezia/amneziawg/awg0.conf` | конфиг сервера |
| `--params` | `/etc/amnezia/amneziawg/params` | параметры установщика |
| `--client-dir` | `/etc/amnezia/amneziawg/clients` | где хранятся конфиги, созданные панелью |
| `--password-hash-file` | `/etc/amnezia/amneziawg/panel.hash` | bcrypt-хеш пароля админа |
| `--tls-cert` / `--tls-key` | — | включают HTTPS |

`awg-panel hash` читает пароль из stdin и печатает bcrypt-хеш (plaintext нигде не хранится).

## Security notes

- Панель запускается под root (нужно для `awg`/`awg-quick`). Не открывай её в интернет
  без необходимости; лучший вариант — доступ через SSH-туннель или доверенную сеть.
- Куки `HttpOnly` + `SameSite=Lax`, флаг `Secure` ставится при HTTPS.
- На формах — CSRF-токен, привязанный к сессии.
- QR/`.conf` отдаются только для клиентов, созданных самой панелью (приватные ключи
  не хранятся на сервере для клиентов, созданных через CLI — это by design WireGuard).

## Architecture

```
cmd/awg-panel/main.go        # flags, TLS, `hash` subcommand
internal/
├── awgctl/                  # control plane (params, peer add/remove, FileController)
├── auth/                    # bcrypt + in-memory sessions + CSRF
├── server/                  # routing, middleware, handlers, rate tracker
└── web/                     # embedded templates + static (htmx, CSS)
```

Pure logic (`awgctl`, `auth`) and HTTP handlers (against a fake `Controller`) are
unit-tested; run `go test ./...`.

## License

MIT — see [../LICENSE](../LICENSE).
