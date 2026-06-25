# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Manage per-client limits (speed, quota, expiry, and enable/disable) from the
  desktop app, not just the web panel. Both are front-ends to the same always-on
  enforcer daemon, so limits apply 24/7 regardless of which one set them. The
  controls appear in the Clients tab only when the web panel is installed.

### Changed
- The lifecycle store is now safe for two processes at once (the panel daemon and
  the `awg-panel client-*` CLI): every change runs under a file lock and re-reads
  first, so neither clobbers the other. The enforcer reconciles speed caps every
  cycle so out-of-band edits take effect promptly.

## [1.2.0] - 2026-06-24

### Changed
- Renamed the project to **AWG Suite** (GitHub repo and Go module path). The old
  `amneziawg-installer` URL redirects, so existing clones, links, and release
  assets keep working.
- Reworded the READMEs and docs for clarity and consistency.

### Added
- Community health files: code of conduct, contributing guide, security policy,
  issue forms, and a pull request template. Enabled private vulnerability reporting.

### Fixed
- Corrected the uninstall menu number in the troubleshooting docs (option 9, not 8).

## [1.1.0] - 2026-06-23

### Added
- Desktop app: sortable client traffic table in Monitoring.
- Desktop app: aggregate traffic over day / week / month in Monitoring.
- Desktop app: a "Server terminal" tab to run commands on the server over SSH.

### Changed
- Renamed the sidebar "Бот" tab to "Telegram Бот".

### Fixed
- Settings: the panel address no longer clips.
- The install log no longer appears at the bottom of every tab.

## [1.0.0] - 2026-06-23

### Added
- Expert mode at install time: obfuscation profiles (mobile, desktop, plain,
  custom) and env-overridable parameters.
- Per-client overrides: split-tunnel `AllowedIPs`, custom DNS, and MTU, from the
  installer, the web panel, and the desktop app.
- Per-field help tooltips for the expert settings.
- `docs/EXPERT.md` and `docs/EXPERT.ru.md`.

### Changed
- Russian is now the primary README; English moved to `README.en.md`.

Defaults are unchanged, so a normal install is byte-identical to before.

## [0.18.0] - 2026-06-22

### Added
- Desktop app: in-app update check, with a banner in Settings and a one-click
  download of the latest release.

## [0.17.0] - 2026-06-22

### Added
- `awg-bot`: a Telegram bot for issuing client profiles, installable and
  removable from the desktop app with an inline setup guide.
- README screenshots for the desktop app and the web panel.

### Security
- Bot access now requires both the admin allowlist and the access password
  (two-factor), instead of either one.

## [0.16.0] - 2026-06-20

### Added
- Desktop app: a Settings tab with server info, connection rename, web-panel
  password change, and a danger zone.
- Web panel: an add-client modal and a Server overview page (host load and
  aggregate traffic).
- A sidebar app-shell layout for the desktop app and the panel.

### Fixed
- Web panel: a cleaner, denser client table with single-line actions.

## [0.15.0] - 2026-06-19

### Added
- Web panel: per-client traffic over day / week / month with sortable columns.

### Changed
- Desktop app: visual refresh (depth, type hierarchy, focus rings).
- Split the docs by language (`docs/*.md` for English, `docs/*.ru.md` for Russian).

### Fixed
- Desktop app: save the client config as a proper `.conf` file.

## [0.14.1] - 2026-06-18

### Fixed
- Desktop app: connect-form inputs disappeared after switching language.

## [0.14.0] - 2026-06-18

### Added
- Desktop app: Russian/English localization with a language switcher, plus custom
  per-server labels.

## [0.13.0] - 2026-06-18

### Added
- VPS rental (referral) links in the desktop app and the README.
- A funding / sponsor link.

## [0.12.1] - 2026-06-17

### Added
- Unit tests for the desktop app's pure logic, wired into CI.

### Fixed
- Windows line endings (CRLF) in the embedded installer script broke `bash -s` on
  the server.
- SSH keepalive so idle sessions are not dropped.
- A macOS first-launch guide for the unsigned app.

## [0.12.0] - 2026-06-17

### Added
- A native desktop app (Wails) for install and client management, built for macOS
  and Windows and attached to releases. Includes saved server profiles, a live
  traffic view, a server health line, remember-password via the OS keychain,
  per-client config/QR on demand, and web-panel integration for advanced limits.

### Fixed
- Replaced unreliable native dialogs with in-app modals and toasts.
- A native Save dialog for the `.conf` file.

## [0.11.1] - 2026-06-17

### Changed
- Note clearly that the QR works only in the AmneziaWG app.

## [0.11.0] - 2026-06-17

### Added
- The installer offers the web panel at the end of a fresh install, plus a
  post-install summary.
- Archive (`.tar.gz` / `.zip`) release assets that preserve the executable bit.

### Fixed
- The release workflow no longer races when attaching assets.

## [0.10.0] - 2026-06-17

### Added
- A language picker at startup.
- The web panel now adopts clients created by the installer or the CLI, so their
  limits can be managed and their configs downloaded.
- Menu options to remove the panel or the monitor when they are installed.

## [0.9.0] - 2026-06-17

### Changed
- The wizard now runs the installer and the management menu directly on the
  server over an interactive SSH session.

## [0.8.0] - 2026-06-17

### Added
- Web panel: edit and rename existing clients.
- Login brute-force lockout (per IP).

### Changed
- More scannable QR codes.

## [0.7.1] - 2026-06-16

### Changed
- `awg-deploy`: clearer server prompt (a bare IP works and defaults to root).

## [0.7.0] - 2026-06-16

### Added
- `awg-deploy`: a guided interactive wizard that runs when launched with no
  arguments.

## [0.6.4] - 2026-06-16

### Added
- `awg-deploy`: an interactive server menu over SSH, and an uninstall command.

## [0.6.3] - 2026-06-16

### Added
- `awg-deploy`: `list` and `remove-client` subcommands, and graceful re-runs on a
  configured server.

## [0.6.2] - 2026-06-16

### Added
- A beginner-friendly install section (per-OS binary table, macOS Gatekeeper steps).

### Changed
- Visible install progress during the DKMS build.

### Fixed
- A scannable PNG QR instead of an oversized terminal QR.

## [0.6.1] - 2026-06-16

### Fixed
- The mobile preset now sets the MTU on the server interface too, not just the client.

## [0.6.0] - 2026-06-16

### Added
- Full English/Russian localization across the docs, the installer UI, and the
  web panel.

## [0.5.0] - 2026-06-16

### Added
- `awg-deploy`: a cross-platform SSH installer, including a Windows `.exe`.
- A non-interactive installer mode for SSH and automation.

## [0.4.0] - 2026-06-16

### Added
- Per-client speed limiting via `tc` (HTB shaping for upload, ingress policing for
  download).

## [0.3.1] - 2026-06-16

### Changed
- Expired clients are disabled rather than deleted, so they can be re-enabled.

## [0.3.0] - 2026-06-16

### Added
- Traffic quotas and time-based expiry, enforced automatically by a background
  reconciler.

## [0.2.0] - 2026-06-16

### Added
- The web management panel (Go + htmx): authentication, a live dashboard, and
  client add/revoke with QR.

### Changed
- Consolidated the tools into a single root Go module (monorepo).

## [0.1.0] - 2026-06-16

### Added
- Initial release: a one-command AmneziaWG installer and client manager for
  Ubuntu/Debian, with a mobile preset (MTU 1280, Jc=3) for reliable mobile
  connections.
- `awg-monitor`: a terminal traffic dashboard.
- Prebuilt Linux amd64/arm64 binaries.

[Unreleased]: https://github.com/hennessyxo/awg-suite/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/hennessyxo/awg-suite/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/hennessyxo/awg-suite/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/hennessyxo/awg-suite/compare/v0.18.0...v1.0.0
[0.18.0]: https://github.com/hennessyxo/awg-suite/compare/v0.17.0...v0.18.0
[0.17.0]: https://github.com/hennessyxo/awg-suite/compare/v0.16.0...v0.17.0
[0.16.0]: https://github.com/hennessyxo/awg-suite/compare/v0.15.0...v0.16.0
[0.15.0]: https://github.com/hennessyxo/awg-suite/compare/v0.14.1...v0.15.0
[0.14.1]: https://github.com/hennessyxo/awg-suite/compare/v0.14.0...v0.14.1
[0.14.0]: https://github.com/hennessyxo/awg-suite/compare/v0.13.0...v0.14.0
[0.13.0]: https://github.com/hennessyxo/awg-suite/compare/v0.12.1...v0.13.0
[0.12.1]: https://github.com/hennessyxo/awg-suite/compare/v0.12.0...v0.12.1
[0.12.0]: https://github.com/hennessyxo/awg-suite/compare/v0.11.1...v0.12.0
[0.11.1]: https://github.com/hennessyxo/awg-suite/compare/v0.11.0...v0.11.1
[0.11.0]: https://github.com/hennessyxo/awg-suite/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/hennessyxo/awg-suite/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/hennessyxo/awg-suite/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/hennessyxo/awg-suite/compare/v0.7.1...v0.8.0
[0.7.1]: https://github.com/hennessyxo/awg-suite/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/hennessyxo/awg-suite/compare/v0.6.4...v0.7.0
[0.6.4]: https://github.com/hennessyxo/awg-suite/compare/v0.6.3...v0.6.4
[0.6.3]: https://github.com/hennessyxo/awg-suite/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/hennessyxo/awg-suite/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/hennessyxo/awg-suite/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/hennessyxo/awg-suite/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/hennessyxo/awg-suite/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/hennessyxo/awg-suite/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/hennessyxo/awg-suite/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/hennessyxo/awg-suite/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/hennessyxo/awg-suite/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/hennessyxo/awg-suite/releases/tag/v0.1.0
