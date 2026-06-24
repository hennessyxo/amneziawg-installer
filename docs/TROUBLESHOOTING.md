# Troubleshooting

[English](TROUBLESHOOTING.md) · [Русский](TROUBLESHOOTING.ru.md)

## The service won't start (`awg-quick@awg0` failed)

```bash
journalctl -u awg-quick@awg0 -n 50
```

- **`Cannot find device "awg0"` / module not loaded**: the kernel module didn't
  build. Make sure the kernel headers are installed and rebuild DKMS:
  ```bash
  apt-get install -y linux-headers-$(uname -r)
  dkms autoinstall
  modprobe amneziawg
  ```
  If the kernel is non-standard (some VPSs), the headers may not match. Update the
  kernel (`apt full-upgrade && reboot`) and run the install again.

- **`RTNETLINK answers: Address already in use`**: the port is taken. Run the
  script, uninstall (menu option 9), and reinstall choosing a different port.

## The client connects but there's no internet

1. Check forwarding:
   ```bash
   sysctl net.ipv4.ip_forward   # should be = 1
   ```
2. Check that the MASQUERADE rule is on the right interface:
   ```bash
   iptables -t nat -L POSTROUTING -n -v
   ```
   The interface in the rule must match `ip route show default`.
3. On the cloud provider side, open the chosen **UDP port** in the
   Security Group / Firewall panel (AWS, Hetzner, Oracle, etc.).

## The connection drops / DPI still blocks it

- The obfuscation parameters **must match** between server and client (except
  `Jc/Jmin/Jmax`). The script guarantees this, so don't edit them by hand.
- Try a different UDP port (e.g. 443 is rarely throttled, but may be taken).
- Some providers throttle all UDP; then you need a TCP wrapper (out of scope for this script).

## `apt` can't find the `amneziawg` package

- **Ubuntu:** make sure the PPA was added:
  ```bash
  add-apt-repository ppa:amnezia/ppa && apt update
  ```
- **Debian:** the PPA targets Ubuntu kernels. If DKMS won't build against your
  Debian kernel, consider the userspace `amneziawg-go` (manual) or use Ubuntu.

## How to see active connections

```bash
awg show awg0
```

A peer's `latest handshake` line shows whether that client is connected.

## Full removal

Run the script → menu option **9** (removes the packages, configs and clients).
