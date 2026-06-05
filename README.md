# Bell Provisioner

Mac utility for factory/lab device provisioning. Guides operators through:

1. **Environment** — gateway URL, VPS vs Mac LAN backend, SSH target, GitHub token
2. **Admin login** — `POST /auth/login` with an account that has `role=admin`
3. **LAN discovery** — mDNS (`raspberrypi.local`), subnet scan, manual IP
4. **Cloud provision (v2)** — select **DTID** + factory **device_id**, then `POST /admin/devices/provision`
5. **Install + verify** — download `doorbell-agent-linux-arm64` CI artifact on the Mac, push to Pi via SSH, install credentials + optional `agent.env`, MQTT smoke test

## Prerequisites

- Mac on the same LAN as the Raspberry Pi
- Pi SSH password auth (`pi@raspberrypi.local`; provisioner does not use Mac SSH keys)
- API gateway with admin routes deployed (`POST /admin/devices/provision`, `GET /admin/device-types`)
- Operator account with `role=admin` in `auth.users` (or listed in gateway `ADMIN_USER_IDS`)
- **Device registry applied** on the target environment before the first v2 unit (see below)
- **pi-streamer** [Agent workflow](https://github.com/sreekumarsh/pi-streamer/blob/main/.github/workflows/agent.yml) has run on `main` (publishes the `doorbell-agent-linux-arm64` artifact)
- Read-only **GitHub token** with access to `sreekumarsh/pi-streamer` and Actions artifacts

### Registry before first v2 unit

v2 provisioning requires device types in the device-service registry. On the VPS (or dev stack), apply the version-controlled YAML **before** factory provisioning:

```bash
# In bell-device-management-service — dry-run first
cd services/device
go run ./tools/device-registry apply --dry-run -f registry.yaml
go run ./tools/device-registry apply -f registry.yaml
```

Equivalent gateway route: `POST /admin/device-registry/apply` (Admin JWT). The provision step loads DTIDs from `GET /admin/device-types`.

See [bell-docs device-service README § Registry CLI](https://github.com/sreekumarsh/bell-docs/blob/main/backend/device-service/README.md#registry-cli) and [device-types-and-capabilities.md](https://github.com/sreekumarsh/bell-docs/blob/main/architecture/device-types-and-capabilities.md).

### Promote an operator to admin

```sql
UPDATE auth.users SET role = 'admin' WHERE phone = '+919876543210';
```

Run migration `005_add_user_role.sql` on the auth database if not already applied.

## v2 factory provision

Operator selects **DTID** (device type from registry) and **device_id** (per-type factory unit number, e.g. `00042`). The app calls:

```http
POST /admin/devices/provision
Authorization: Bearer <admin_jwt>

{
  "dtid": "dt_8f3k2m9x1p",
  "device_id": "00042",
  "serial_number": "DB-2605-0042",
  "hardware_version": "1.1",
  "public_key_pem": "-----BEGIN PUBLIC KEY-----\n...",
  "overwrite": true
}
```

Response includes `global_device_id`, `dsid`, and MQTT credentials. Install writes `/etc/doorbell/identity.json` on the Pi:

```json
{
  "global_device_id": "dt_8f3k2m9x1p_00042",
  "device_id": "00042",
  "dtid": "dt_8f3k2m9x1p",
  "dsid": "ds_7q2w9e4r",
  "hardware_version": "1.1",
  "mqtt_username": "dt_8f3k2m9x1p_00042",
  "mqtt_password": "..."
}
```

`mqtt_username` equals `global_device_id` for MQTT topics and agent `CloudDeviceID()`. Label QR should encode **DSID** for claim.

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

Saved to `~/Library/Application Support/bell-provisioner/config.json` (`chmod 0600`). SSH password stays session-only; `github_token` can be saved there for factory use.

| Setting | Default |
|---------|---------|
| Gateway URL | `https://api.vyooham.com` |
| Backend profile | VPS |
| SSH | `pi@raspberrypi.local:22` |
| Agent repo | `git@github.com:sreekumarsh/pi-streamer.git` (used to resolve GitHub owner/repo) |
| Agent artifact | `doorbell-agent-linux-arm64` |

## Agent install flow (CI artifact)

Matches **pi-streamer** workflow `.github/workflows/agent.yml`:

1. On push to `main` (under `agent/**`), CI builds `linux/arm64` `doorbell-agent`, packages `pi-release/` (binary, `doorbell-agent.service`, `VERSION`) into `doorbell-agent-linux-arm64.tar.gz`, and uploads artifact **`doorbell-agent-linux-arm64`**.
2. **bell-provisioner** (on your Mac) calls the GitHub API with your token, downloads the latest artifact zip, and extracts the binary + systemd unit.
3. Over SSH, the app uploads files to `/tmp` on the Pi, runs `sudo install` to `/usr/local/bin/doorbell-agent` and `/etc/systemd/system/`, installs **go2rtc** if missing, then deploys `/etc/doorbell/` credentials and restarts the service.

### GitHub token permissions

**Fine-grained PAT** (recommended):

- Repository: `sreekumarsh/pi-streamer`
- **Contents:** Read-only
- **Actions:** Read-only (to download workflow artifacts)

**Classic PAT:** `repo` scope (private repository).

Paste in **Environment → GitHub token** (stored in `config.json` when you continue), or set `"github_token"` in that file directly.

## Related docs

- [pi-streamer Agent workflow](https://github.com/sreekumarsh/pi-streamer/blob/main/.github/workflows/agent.yml)
- [device-types-and-capabilities.md](https://github.com/sreekumarsh/bell-docs/blob/main/architecture/device-types-and-capabilities.md) — v2 IDs and provision flow
- [go-agent.md § identity.json](https://github.com/sreekumarsh/bell-docs/blob/main/firmware/go-agent.md) — on-device identity format
- [provisioning-utility.md](https://github.com/sreekumarsh/bell-docs/blob/main/operations/provisioning-utility.md) — operator guide
- [firmware/vps-agent-setup.md](https://github.com/sreekumarsh/bell-docs/blob/main/firmware/vps-agent-setup.md) — VPS agent + provisioning context
- [backend/api-gateway/README.md](https://github.com/sreekumarsh/bell-docs/blob/main/backend/api-gateway/README.md) — admin routes

## Future

- v2.1: factory HTTP endpoint on device (no SSH)
- v3: USB credential pipe
