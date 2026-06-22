# awg-bot

[English](BOT.md) · [Русский](BOT.ru.md)

> Access-controlled **Telegram bot** for issuing AmneziaWG client profiles — Go, single binary.

![go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)
![telegram](https://img.shields.io/badge/Telegram-Bot%20API-26A5E4?logo=telegram)

Hand out VPN profiles from a Telegram chat: send `/new phone` and the bot replies
with the ready `.conf` file and a QR. It runs on the server next to the VPN and
reuses the same `awgctl` core as the web panel.

## Features

- 💬 **Commands**: `/new <name>` (creates a client, sends `.conf` + QR), `/list`,
  `/config <name>` (resend), `/revoke <name>`, plus `/start` and `/help`.
- 🔐 **Access control (two factors)**: a user must be **both** on the **admin
  allowlist** (Telegram IDs) **and** have entered the **access password** via
  `/auth <password>`. A non-listed user is rejected even with the right password;
  removing an ID from the allowlist revokes access immediately. Passing the
  password is remembered across restarts.
- 📡 **No inbound port**: the bot **long-polls** the Telegram API, so nothing needs
  to be exposed on the server — works behind any firewall/NAT.
- 📦 **Single binary**: no extra services; reuses the installer's config + client dir.

## Get a bot token

1. Open [@BotFather](https://t.me/BotFather) in Telegram → `/newbot` → follow the
   prompts. It gives you a **token** like `123456:ABC-DEF...`.
2. (Optional) To preset admins, get your numeric Telegram ID from
   [@userinfobot](https://t.me/userinfobot).

## Install

Via the installer menu (recommended):

```bash
sudo bash amneziawg-install.sh   # → option 8 "Telegram bot"
```

It downloads the binary, asks for the **token** and an **access password**, then
starts a systemd service. Non-interactive (automation / SSH):

```bash
AWG_BOT_TOKEN='123456:ABC...' \
AWG_BOT_ADMINS='12345678,87654321' \
AWG_BOT_PASSWORD='Admin2@' \
  sudo -E bash amneziawg-install.sh --install-bot
sudo bash amneziawg-install.sh --remove-bot
```

Both the allowlist (`AWG_BOT_ADMINS`) and the password are required. Then, from
an allowlisted account in Telegram: `/auth <password>` once, then `/new laptop`.

## Flags

| Flag | Default | Meaning |
|------|---------|---------|
| `--token-file` | — | file with the bot token (preferred over `--token`) |
| `--admins` | — | comma-separated Telegram user IDs always allowed |
| `--password-hash-file` | — | bcrypt hash of the access password (`awg-bot hash`) |
| `--auth-store` | `/etc/amnezia/amneziawg/bot-authorized.json` | remembered authorized chats |
| `--iface` | `awg0` | AmneziaWG interface |
| `--conf` / `--params` / `--client-dir` / `--store` | installer paths | server config + client data |
| `--lang` | `ru` | bot reply language (`ru`/`en`) |

`awg-bot hash` reads a password from stdin and prints a bcrypt hash (the plaintext
is never stored). **Both** `--admins` and `--password-hash-file` are required — a
user must be on the allowlist *and* pass the password.

## Security notes

- The **token** is a secret — it is stored `600` under `umask 077`, never in the repo.
- Access is **two-factor**: the allowlist (who) **and** the password (proof). The
  bot answers management commands only when both are satisfied; everyone else gets
  "not authorized". Set a strong access password (same complexity rule as the
  panel: lower- and upper-case, a digit and a special character).
- When a user sends `/auth <password>`, the bot **deletes that message** so the
  password doesn't linger in the chat history.
- The bot runs as root (needed for `awg`/`awg-quick`), like the panel.

## Architecture

```
cmd/awg-bot/main.go      # flags, wiring, `hash` subcommand
internal/tgbot/
├── api.go               # minimal Telegram Bot API client (long polling, multipart upload)
├── auth.go              # admin IDs + bcrypt password + persisted authorized chats
├── command.go           # pure command parsing
├── bot.go               # poll loop + command dispatch
└── i18n.go              # RU/EN replies
```

It reuses `internal/awgctl` (add/remove/list/config) and `internal/auth` (bcrypt).
Pure logic (command parsing, authorization) is unit-tested with `go test ./...`.

## License

MIT — see [../LICENSE](../LICENSE).
