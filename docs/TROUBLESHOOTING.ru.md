# Решение проблем

[English](TROUBLESHOOTING.md) · [Русский](TROUBLESHOOTING.ru.md)

## Служба не запускается (`awg-quick@awg0` failed)

```bash
journalctl -u awg-quick@awg0 -n 50
```

- **`Cannot find device "awg0"` / модуль не загружен** — модуль ядра не собрался.
  Проверьте, что установлены заголовки ядра, и пересоберите DKMS:
  ```bash
  apt-get install -y linux-headers-$(uname -r)
  dkms autoinstall
  modprobe amneziawg
  ```
  Если ядро нестандартное (некоторые VPS), заголовки могут не совпадать —
  обновите ядро (`apt full-upgrade && reboot`) и запустите установку заново.

- **`RTNETLINK answers: Address already in use`** — порт занят. Запустите скрипт,
  удалите установку (пункт меню 8) и поставьте заново, выбрав другой порт.

## Клиент подключается, но нет интернета

1. Проверьте forwarding:
   ```bash
   sysctl net.ipv4.ip_forward   # должно быть = 1
   ```
2. Проверьте, что MASQUERADE-правило на правильном интерфейсе:
   ```bash
   iptables -t nat -L POSTROUTING -n -v
   ```
   Интерфейс в правиле должен совпадать с `ip route show default`.
3. На стороне облачного провайдера откройте выбранный **UDP-порт**
   в Security Group / Firewall панели (AWS, Hetzner, Oracle и т.д.).

## Подключение рвётся / DPI всё равно блокирует

- Параметры обфускации **должны совпадать** у сервера и клиента
  (кроме `Jc/Jmin/Jmax`). Скрипт это гарантирует — не редактируйте вручную.
- Попробуйте другой UDP-порт (например, 443 редко режут, но он может быть занят).
- Некоторые провайдеры режут весь UDP — тогда нужна TCP-обёртка (вне рамок скрипта).

## `apt` не находит пакет `amneziawg`

- **Ubuntu:** убедитесь, что PPA добавился:
  ```bash
  add-apt-repository ppa:amnezia/ppa && apt update
  ```
- **Debian:** PPA собирается под ядра Ubuntu. Если DKMS не собирается под ваше
  ядро Debian, рассмотрите userspace-вариант `amneziawg-go` (вручную) или
  используйте Ubuntu.

## Как посмотреть активные подключения

```bash
awg show awg0
```

Строка `latest handshake` у пира показывает, подключён ли клиент.

## Полное удаление

Запустите скрипт → пункт меню **8** (удалит пакеты, конфиги и клиентов).
