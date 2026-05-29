# Bell Provisioner

Mac utility for factory/lab device provisioning. Guides operators through:

1. **Environment** — gateway URL, VPS vs Mac LAN backend, SSH target, GitHub token
2. **Admin login** — `POST /auth/login` with an account that has `role=admin`
3. **LAN discovery** — mDNS (`raspberrypi.local`), subnet scan, manual IP
4. **Cloud provision** — `POST /admin/devices/provision` with optional `overwrite: true` to re-provision an existing serial (new keys + MQTT password)
5. **Install + verify** — download `doorbell-agent-linux-arm64` CI artifact on the Mac, push to Pi via SSH, install credentials + optional `agent.env`, MQTT smoke test

## Prerequisites

- Mac on the same LAN as the Raspberry Pi
- Pi SSH password auth (`pi@raspberrypi.local`; provisioner does not use Mac SSH keys)
- API gateway with admin routes deployed (`POST /admin/devices/provision`)
- Operator account with `role=admin` in `auth.users` (or listed in gateway `ADMIN_USER_IDS`)
- **pi-streamer** [Agent workflow](https://github.com/sreekumarsh/pi-streamer/blob/main/.github/workflows/agent.yml) has run on `main` (publishes the `doorbell-agent-linux-arm64` artifact)
- Read-only **GitHub token** with access to `sreekumarsh/pi-streamer` and Actions artifacts

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
- [firmware/vps-agent-setup.md](https://github.com/sreekumarsh/bell-docs/blob/main/firmware/vps-agent-setup.md) — VPS agent + provisioning context
- [backend/api-gateway/README.md](https://github.com/sreekumarsh/bell-docs/blob/main/backend/api-gateway/README.md) — admin routes

## Future

- v2: factory HTTP endpoint on device (no SSH)
- v3: USB credential pipe
