# Bell Provisioner

Mac utility for factory/lab device provisioning. Guides operators through:

1. **Environment** — gateway URL, VPS vs Mac LAN backend, SSH target
2. **Admin login** — `POST /auth/login` with an account that has `role=admin`
3. **LAN discovery** — mDNS (`raspberrypi.local`), subnet scan, manual IP
4. **Cloud provision** — `POST /admin/devices/provision` (RSA keygen on Mac, public key to auth)
5. **Install + verify** — SCP credentials to Pi via SSH, optional `agent.env`, MQTT smoke test

## Prerequisites

- Mac on the same LAN as the Raspberry Pi
- Pi SSH access (`pi@raspberrypi.local` or key in `~/.ssh/id_ed25519`)
- API gateway with admin routes deployed (`POST /admin/devices/provision`)
- Operator account with `role=admin` in `auth.users` (or listed in gateway `ADMIN_USER_IDS`)

### Promote an operator to admin

```sql
UPDATE auth.users SET role = 'admin' WHERE phone = '+919876543210';
```

Run migration `005_add_user_role.sql` on the auth database if not already applied.

## Build

```bash
# Requires Go 1.23+, Node.js, Wails v2
go install github.com/wailsapp/wails/v2/cmd/wails@latest
cd ~/Workspace/bell-provisioner
wails build
```

Output: `build/bin/bell-provisioner.app`

Dev mode:

```bash
wails dev
```

## Configuration

Saved to `~/Library/Application Support/bell-provisioner/config.json` (no passwords stored).

| Setting | Default |
|---------|---------|
| Gateway URL | `https://api.vyooham.com` |
| Backend profile | VPS |
| SSH | `pi@raspberrypi.local:22` |

## Related docs

- [firmware/vps-agent-setup.md](../bell-docs/firmware/vps-agent-setup.md) — VPS agent + provisioning context
- [backend/api-gateway/README.md](../bell-docs/backend/api-gateway/README.md) — admin routes

## Future

- v2: factory HTTP endpoint on device (no SSH)
- v3: USB credential pipe
