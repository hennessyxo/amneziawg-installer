# Security Policy

AWG Suite sets up and manages a VPN, so security matters here. Thanks for helping
keep it safe.

## Supported versions

This is an actively developed project. Security fixes land on `main` and ship in the
next release. Please test against the [latest release](https://github.com/hennessyxo/awg-suite/releases/latest)
before reporting.

## Reporting a vulnerability

Please report security issues privately, not in a public issue or pull request.

Use GitHub's private vulnerability reporting: open the repository's **Security** tab
and click **Report a vulnerability**. That opens a private channel visible only to
the maintainer.

When you report, include where you can:

- which component is affected (installer, `awg-panel`, `awg-bot`, `awg-deploy`, the
  GUI, or a shared package),
- the version or commit,
- steps to reproduce or a proof of concept,
- the impact as you see it.

You can expect an acknowledgement within a few days. Once a fix is ready, it ships in
a release and the report is credited unless you prefer otherwise.

## Scope

Things that are in scope and worth reporting:

- authentication or session handling in the web panel,
- access control in the Telegram bot (the allowlist plus password),
- SSH host-key handling and credential storage in `awg-deploy` and the GUI,
- command construction that could allow injection on the server,
- secret handling (keys, password hashes, tokens) and file permissions.

Out of scope: vulnerabilities in AmneziaWG/WireGuard themselves (report those
upstream), and issues that require an already-compromised server or physical access.

## Good to know

- Private keys, parameters, and the panel password hash are stored with `600`
  permissions under `umask 077`.
- The web panel and bot run as root because `awg`/`awg-quick` require it. Do not
  expose the panel to the public internet without need; an SSH tunnel is documented
  in [docs/PANEL.md](docs/PANEL.md).
