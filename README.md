# Bell Provisioner

Mac utility for factory/lab device provisioning. After **admin login**, the dashboard offers:

- **Device registry** — manage capabilities, families, and device types via gateway `/admin/*` routes
- **Start provisioning** — guided flow: environment → LAN discovery → cloud provision (v2) → SSH install + MQTT verify

## Flow

1. **Login** — gateway URL + admin credentials (`POST /auth/login`, `role=admin`)
2. **Dashboard** — choose registry management or start a provisioning run
3. **Provisioning** (optional wizard):
   - **Environment** — VPS vs Mac LAN backend, SSH target, GitHub token
   - **Discover** — mDNS / subnet scan / manual IP
   - **Provision (v2)** — select **DTID** + factory **device_id**, then `POST /admin/devices/provision`
   - **Install + verify** — resolve install profile from DTID (doorbell vs NVR), deploy matching agent + deps, MQTT smoke test
4. **Registry** (from dashboard) — `GET/POST/PATCH /admin/capabilities`, `/admin/device-families`, `/admin/device-types`, and `POST /admin/device-registry/apply`

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
| Agent repo (doorbell) | `git@github.com:sreekumarsh/pi-streamer.git` |
| Agent artifact (doorbell) | `doorbell-agent-linux-arm64` |
| NVR agent repo | `git@github.com:sreekumarsh/vyooham-nvr.git` (`main`) |
| Sense agent repo | `git@github.com:sreekumarsh/vyooham-sense.git` (`main`) |
| Sense MQTT transport | `plain` (`sense_mqtt_transport`: `plain` \| `mtls`) |

## Install profiles (per device type)

Install resolves the provisioned **DTID** via `GET /admin/device-types` and picks a profile by **family** (with capability fallback):

| Registry | Profile | On-device | Agent | Runtime deps |
|----------|---------|-----------|-------|--------------|
| Family `df_door0001` (or `cap_cam00001`) | **doorbell** | `/etc/doorbell/` | `doorbell-agent` (CI artifact from pi-streamer) | ffmpeg, v4l-utils, go2rtc, motion/ONNX |
| Family `df_sense` (or `cap_npu00015`) | **sense** | `/etc/vyooham-sense/` | `control-agent` (built from vyooham-sense git on Mac, linux/arm64) | none for control-agent (no camera stack) |
| Family `df_nvr0001` (or `cap_mcr00012` / `cap_lan00014`) | **nvr** | `/etc/vyooham/` | `control-agent` (built from vyooham-nvr git on Mac, linux/amd64) | none for control-agent (no camera stack) |

Seed types: `dt_wired0001` (doorbell), `dt_nvr0001` (Vyooham NVR v1), `dt_sense_v1` (Sense V1).

`cap_npu00015` is checked **before** `cap_lan00014`: `dt_sense_v1` declares both, and
matching LAN relay first would install a Sense box as an NVR — writing its identity and
mTLS material to `/etc/vyooham/`, where the Sense agent never looks.

### Sense MQTT transport (`sense_mqtt_transport`)

Sense has no installed base, so it is the one product that could ship on mTLS `8883`
from day one rather than Phase A's plain `1883`. It currently does **not**, because both
preconditions are unmet:

1. `mqtt.vyooham.com` has no DNS A record (NXDOMAIN) — the box could not resolve its broker.
2. auth-service runs without `MQTT_CA_CERT_FILE` / `MQTT_CA_KEY_FILE`, so its `mqttca`
   signer is nil and provision responses omit `device_crt` / `ca_crt` (both `omitempty`).

Set `"sense_mqtt_transport": "mtls"` in `config.json` to provision
`ssl://mqtt.vyooham.com:8883` + `MQTT_TLS_ENABLED=true`; flip
`config.DefaultSenseTransport` to make it the default once both hold. Install **aborts
before touching the device** if a TLS env would be written without both certs present.

## Doorbell agent install (CI artifact)

Matches **pi-streamer** workflow `.github/workflows/agent.yml`:

1. On push to `main` (under `agent/**`), CI builds `linux/arm64` `doorbell-agent`, packages `pi-release/` (binary, `doorbell-agent.service`, `VERSION`) into `doorbell-agent-linux-arm64.tar.gz`, and uploads artifact **`doorbell-agent-linux-arm64`**.
2. **bell-provisioner** (on your Mac) calls the GitHub API with your token, downloads the latest artifact zip, and extracts the binary + systemd unit.
3. Over SSH, the app uploads files to `/tmp` on the Pi, runs `sudo install` to `/usr/local/bin/doorbell-agent` and `/etc/systemd/system/`, installs **doorbell OS runtime dependencies** (see below), then deploys `/etc/doorbell/` credentials and restarts the service.

### Doorbell OS packages (Install + verify)

Before the agent restarts, provisioner runs an idempotent apt + binary step over SSH (same sudo password as login). Only missing items are installed:

| Item | Purpose |
|------|---------|
| `ffmpeg` | go2rtc exec producer, local NVR recording, motion snapshots |
| `v4l-utils` | Camera detection (`v4l2-ctl`, agent startup hints) |
| `curl`, `ca-certificates` | Download go2rtc release binary |
| `/usr/local/bin/go2rtc` | Streaming (latest GitHub release, arm64/amd64; override with `GO2RTC_VERSION` on Pi) |
| `/var/lib/doorbell/venv` | Python venv with `onnxruntime`, `pillow`, `numpy` for motion classification |
| `/var/lib/doorbell/models/yolov8n.onnx` | YOLOv8n ONNX model (embedded in provisioner, uploaded to Pi) |
| `/usr/local/bin/motion-classify.py` | Motion inference script (embedded from pi-streamer) |

Provisioner also deploys `agent.env` (when enabled) with `MOTION_*` paths matching the above. After install, it polls `http://<device>:4444/setup/identity` until the agent setup server responds — the device is ready for QR claim when `claimed:false` in identity.json and setup server is up.

Re-running Install on the same device is safe: existing packages and go2rtc are detected and skipped. If apt fails (no network, wrong sudo password), the UI reports the error.

## NVR agent install (git build)

There is no Actions artifact for `control-agent` yet. For the **nvr** profile:

1. Download the latest `vyooham-nvr` tarball from GitHub (`main`).
2. On the Mac, `GOOS=linux GOARCH=amd64 go build` `services/control-agent/cmd/control-agent`.
3. Over SSH, install binary + embedded systemd unit, write credentials to `/etc/vyooham/`, deploy NVR `agent.env` (MQTT + identity paths only — no camera/GPIO/motion), restart `control-agent`, poll setup `:4444`.

CLI deps-only helper: `go run ./cmd/install-deps --profile=doorbell|nvr|sense`.

### GitHub token permissions

**Fine-grained PAT** (recommended):

- Repositories: `sreekumarsh/pi-streamer` and `sreekumarsh/vyooham-nvr`
- **Contents:** Read-only
- **Actions:** Read-only (doorbell CI artifacts)

**Classic PAT:** `repo` scope (private repositories).

Paste in **Environment → GitHub token** (stored in `config.json` when you continue), or set `"github_token"` in that file directly.

## Related docs

- [pi-streamer Agent workflow](https://github.com/sreekumarsh/pi-streamer/blob/main/.github/workflows/agent.yml)
- [vyooham-nvr control-agent](https://github.com/sreekumarsh/vyooham-nvr/blob/main/services/control-agent/README.md) — NVR identity path `/etc/vyooham/`
- [device-types-and-capabilities.md](https://github.com/sreekumarsh/bell-docs/blob/main/architecture/device-types-and-capabilities.md) — v2 IDs and provision flow
- [go-agent.md § identity.json](https://github.com/sreekumarsh/bell-docs/blob/main/firmware/go-agent.md) — on-device identity format
- [provisioning-utility.md](https://github.com/sreekumarsh/bell-docs/blob/main/operations/provisioning-utility.md) — operator guide
- [firmware/vps-agent-setup.md](https://github.com/sreekumarsh/bell-docs/blob/main/firmware/vps-agent-setup.md) — VPS agent + provisioning context
- [backend/api-gateway/README.md](https://github.com/sreekumarsh/bell-docs/blob/main/backend/api-gateway/README.md) — admin routes

## Future

- v2.1: factory HTTP endpoint on device (no SSH)
- v3: USB credential pipe
- Switch NVR install to CI artifacts once vyooham-nvr publishes them (same pattern as doorbell)