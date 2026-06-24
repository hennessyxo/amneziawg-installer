# Expert mode

[English](EXPERT.md) · [Русский](EXPERT.ru.md)

> The tool is zero-config for normal users. If you know what you're doing, every
> knob below is optional and defaults to the safe, well-tested values, so a
> plain install is unchanged.

There are two kinds of expert settings:

1. **Server-wide**: chosen once at install (obfuscation profile/params, port, DNS).
2. **Per-client**: chosen when you add a client (split tunnel, DNS, MTU).

---

## 1. Obfuscation profile (install time)

AmneziaWG hides the WireGuard handshake behind junk packets and custom headers.
Pick a profile when installing:

| Profile | What it does |
|---------|--------------|
| `mobile` | **Default.** MTU 1280, Jc=3, gentle junk. Reliable on 4G/LTE and PC alike. |
| `desktop` | Higher MTU (1420), slightly more junk. For broadband/PC links. |
| `plain` | **Plain WireGuard**: junk/headers off, interoperable with standard WG. |
| `custom` | You set the individual parameters yourself (see below). |

**Where to choose it:**
- **Desktop app** → install screen → *Advanced* → *Obfuscation profile*.
- **Installer (interactive / `awg-deploy`)** → it asks during install (Enter = mobile).
- **Non-interactive / scripting** → `AWG_PRESET=plain sudo -E bash amneziawg-install.sh --yes`.

### Individual parameters (env, install time)

```bash
AWG_PRESET=custom \
AWG_JC=2 AWG_JMIN=40 AWG_JMAX=90 \
AWG_S1=100 AWG_S2=140 \
AWG_H1=… AWG_H2=… AWG_H3=… AWG_H4=… \
AWG_MTU=1380 \
sudo -E bash amneziawg-install.sh --yes
```

Any value you don't set keeps the profile's default (randomized for `Jmin/Jmax/S1/S2/H1-H4`).

> ⚠️ **`S1`, `S2`, `H1`–`H4` must be identical on the server and every client.** They
> are baked into each client's `.conf` at creation, so if you change them later you
> must **re-issue all client configs**. `Jc/Jmin/Jmax` may differ per peer.

## 2. Port & DNS (install time)

- **Port**: `AWG_PORT=51820` (blank = a free random UDP port is picked). Also in the
  desktop app's *Advanced*, and the installer prompt.
- **DNS**: `AWG_DNS1=1.1.1.1 AWG_DNS2=1.0.0.1` (server default for new clients). Also in
  the desktop app's *Advanced*, and the installer prompt.

## 3. Per-client overrides (when adding a client)

Each client can override the defaults. This is useful for **split tunnel** (route only
some subnets through the VPN), a custom resolver, or a different MTU.

| Field | Default | Example |
|-------|---------|---------|
| **Routes** (`AllowedIPs`) | `0.0.0.0/0,::/0` (full tunnel) | `10.0.0.0/8,192.168.0.0/16` (split) |
| **DNS** | server default | `9.9.9.9` |
| **MTU** | server default | `1380` |

**Where to set them:**
- **Web panel** → *Add client* → *Advanced*.
- **Desktop app** → Clients tab → *Advanced client settings*.
- **Installer / `--add-client`** (env):
  ```bash
  AWG_ALLOWED_IPS='10.0.0.0/8,192.168.0.0/16' AWG_CLIENT_DNS='9.9.9.9' \
    sudo -E bash amneziawg-install.sh --add-client work-laptop
  ```

Routes (`AllowedIPs`) only change what **that client** sends through the tunnel. They
don't affect server routing or addressing, so they're safe to tweak per client.

## Examples

**Plain WireGuard server** (no obfuscation, e.g. for a trusted network):
```bash
AWG_PRESET=plain sudo -E bash amneziawg-install.sh --yes
```

**A client that only routes a corporate subnet** (everything else stays direct):
```bash
AWG_ALLOWED_IPS='10.20.0.0/16' sudo -E bash amneziawg-install.sh --add-client office
```

## Caveats

- `S1/S2/H1-H4` must match server ↔ client (see the warning above).
- `plain` mode removes the DPI-evading obfuscation, so only use it where censorship
  isn't a concern.
- Bad per-client values are ignored (the safe default is used) rather than written
  into a config.

## Not yet configurable

- **Custom VPN subnet** (`10.66.66.0/24`): on the roadmap. It's woven through client
  IP allocation, so it needs careful, tested work before exposing.

## License

MIT, see [../LICENSE](../LICENSE).
