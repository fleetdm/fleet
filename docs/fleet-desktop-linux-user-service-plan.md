# Fleet Desktop Linux User Service - Implementation Plan

**Issue:** [#33432](https://github.com/fleetdm/fleet/issues/33432) / [#39110](https://github.com/fleetdm/fleet/issues/39110)

## Problem

Fleet Desktop on Linux is currently launched by orbit (running as root) via `sudo runuser -u <user> -- env ... fleet-desktop`. This approach has caused many issues:
- Browser launching fails because the process doesn't fully inherit the user's environment, SELinux context, etc.
- Orbit must detect the GUI user, session type (X11/Wayland), display variable — all fragile heuristics
- Multiple bugs filed: #31087, #29793, #29654, #25924, #34501, #34303, #36024

## Solution

Install Fleet Desktop as a **systemd user service** (`--user`) that runs in the user's own session, inheriting their full environment naturally.

## Current architecture

```
systemd (system) → orbit.service (root)
                      └─ sudo runuser -u <user> -- env DISPLAY=:0 ... fleet-desktop
```

Orbit's `desktopRunner` in `orbit/cmd/orbit/orbit.go`:
1. Polls for a GUI user via `loginctl` every 30s
2. Detects session type (X11/Wayland) and display variable
3. Launches `fleet-desktop` via `sudo runuser` with manually constructed env vars
4. Monitors the process every 15s, restarts if it dies

## Proposed architecture

```
systemd (system) → orbit.service (root)
                      └─ writes /opt/orbit/desktop.env (env vars for desktop)

systemd (user)   → fleet-desktop.service (per-user, auto-started on login)
                      └─ ConditionPathExists=/opt/orbit/desktop.env
                      └─ EnvironmentFile=/opt/orbit/desktop.env
                      └─ ExecStart=/opt/orbit/bin/desktop/fleet-desktop (symlink)
```

## Key design decisions

### 1. Systemd user service unit file

Install to `/usr/lib/systemd/user/fleet-desktop.service` during packaging. This path is used by systemd for vendor-provided user units. Using `systemctl --global enable` makes it start for ALL users on login.

```ini
[Unit]
Description=Fleet Desktop
After=graphical-session.target
PartOf=graphical-session.target
# Don't start until orbit has written the environment file.
ConditionPathExists=/opt/orbit/desktop.env

[Service]
Type=simple
EnvironmentFile=/opt/orbit/desktop.env
ExecStart=/opt/orbit/bin/desktop/fleet-desktop
Restart=on-failure
RestartSec=5

[Install]
WantedBy=graphical-session.target
```

Key choices:
- **`graphical-session.target`**: ensures desktop only starts when a graphical session is active — no more manual GUI user detection
- **`ConditionPathExists`**: prevents the service from starting before orbit writes the env file (avoids errors on first install)
- **`EnvironmentFile`**: orbit (root) writes config that user service reads — clean IPC boundary
- **`Restart=on-failure`**: systemd handles restarts — no more orbit polling
- **`PartOf=graphical-session.target`**: stops when user logs out

### 2. Environment file for configuration passing

Orbit writes `/opt/orbit/desktop.env` with the env vars Fleet Desktop needs. This file is readable by all users (0644). It contains:
- `FLEET_DESKTOP_FLEET_URL`
- `FLEET_DESKTOP_DEVICE_IDENTIFIER_PATH`
- `FLEET_DESKTOP_FLEET_TLS_CLIENT_CERTIFICATE`
- `FLEET_DESKTOP_FLEET_TLS_CLIENT_KEY`
- `FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST`
- `FLEET_DESKTOP_TUF_UPDATE_ROOT`
- `FLEET_DESKTOP_FLEET_ROOT_CA` (if set)
- `FLEET_DESKTOP_INSECURE` (if set)

Orbit refreshes this file every 30s and ensures token permissions stay correct.

### 3. Token file permissions

The token file at `/opt/orbit/identifier` is currently only readable by root (written by orbit). We need to make it readable by the user running desktop. Change permissions to `0644` so the user service can read it. Orbit periodically checks and corrects these permissions.

### 4. Orbit changes (what gets removed / changed)

**In orbit's main loop (`orbit.go`):**
- On Linux, when `--fleet-desktop` is set, instead of creating a `desktopRunner`, orbit:
  1. Writes `/opt/orbit/desktop.env` with the required env vars
  2. Ensures the token file has proper permissions (0644)
  3. Calls `systemctl --user daemon-reload` + `restart` for logged-in users via `runuser`

**The `desktopRunner` stays** for macOS/Windows (no changes to those platforms). On Linux, it's replaced by a simpler `desktopUserServiceManager` that just manages the env file and service state.

### 5. Packaging changes (`linux_shared.go`)

- Add `fleet-desktop.service` to `/usr/lib/systemd/user/` in the package
- Add a stable symlink `/opt/orbit/bin/desktop/fleet-desktop` → versioned binary
- **postinstall**: `systemctl --global enable fleet-desktop.service` (no explicit start — orbit handles that after writing the env file)
- **preremove**: `systemctl --global disable fleet-desktop.service` + stop for logged-in users
- **postremove**: clean up `/usr/lib/systemd/user/fleet-desktop.service` and `/opt/orbit`

### 6. Migration from old to new (upgrade path)

When upgrading from a package that uses the old `desktopRunner` approach:
- The new orbit binary won't launch desktop via `execuser.Run` on Linux anymore
- The postinstall script enables the user service
- The old `pkill fleet-desktop` in preremove handles killing the old process
- On next login, systemd starts the user service automatically

### 7. Desktop binary symlink

Add a stable symlink for the service to reference:
`/opt/orbit/bin/desktop/fleet-desktop` → actual versioned binary (e.g. `.../linux-arm64/stable/fleet-desktop/fleet-desktop`)

This avoids hardcoding channel/platform paths in the service unit file.

## Files modified

| File | Change |
|------|--------|
| `orbit/cmd/orbit/orbit.go` | On Linux, use `desktopUserServiceManager` instead of `desktopRunner` |
| `orbit/pkg/packaging/linux_shared.go` | Add user service unit file, desktop symlink, update post/pre/posttrans scripts |
| `orbit/cmd/orbit/desktop_service_linux.go` | New file: `desktopUserServiceManager` implementation |
| `orbit/cmd/orbit/desktop_service_other.go` | New file: build tag stub for non-Linux |

## Files NOT modified (kept as-is)

| File | Reason |
|------|--------|
| `orbit/pkg/execuser/execuser_linux.go` | Still used by other `execuser` callers (scripts, etc.) |
| `orbit/pkg/user/user_linux.go` | Still used for other user detection needs |
| `orbit/cmd/desktop/desktop.go` | Desktop binary itself is unchanged — it reads env vars the same way |
| `orbit/cmd/desktop/desktop_linux.go` | Unchanged — tray icon behavior stays the same |

## Risks and considerations

1. **Multi-user systems**: `systemctl --global enable` starts for ALL users. If multiple GUI users are logged in, each gets their own fleet-desktop instance. The current approach only runs one instance for the first detected user. This is actually the *correct* behavior.

2. **Token file sharing**: The identifier token is per-device, not per-user. Multiple user services will share the same token file. This should work fine since desktop only reads it.

3. **Headless users**: The `graphical-session.target` dependency ensures the service only starts for users with a graphical session. SSH-only users won't get fleet-desktop.

4. **Desktop environment compatibility**: By running as a true user service, we inherit `DISPLAY`, `WAYLAND_DISPLAY`, `DBUS_SESSION_BUS_ADDRESS`, etc. naturally from the user session — eliminating the entire class of display detection bugs.

5. **Older distros without user systemd**: Some minimal distros may not have user session support. We should detect this at install time and fall back (or document it as a requirement).

6. **TUF updates**: When orbit updates the desktop binary, orbit restarts itself. The new orbit calls `restartServiceForLoggedInUsers()` on startup, which restarts the user service with the updated binary.

## Test plan

### 1. Installation & Service Enablement

**Note:** `fleetctl package` downloads the orbit binary from TUF, so it won't include local orbit code changes. To test the new `desktopUserServiceManager` code path:

1. Cross-compile orbit locally: `GOOS=linux GOARCH=arm64 go build -o ./build/orbit-linux-arm64 ./orbit/cmd/orbit`
2. Copy it to the target machine (e.g. `scp ./build/orbit-linux-arm64 user@host:~/`)
3. Stop orbit: `sudo systemctl stop orbit`
4. Replace the binary: `sudo cp ~/orbit-linux-arm64 /opt/orbit/bin/orbit/linux-arm64/stable/orbit`
5. Disable TUF updates (otherwise TUF reverts the binary on startup): `echo 'ORBIT_DISABLE_UPDATES=true' | sudo tee -a /etc/default/orbit`
6. Start orbit: `sudo systemctl start orbit`
7. Verify the new code path: `journalctl -u orbit --since "1 min ago" | grep -i desktop` — should show `"managing fleet-desktop as systemd user service"` and `"restarted fleet-desktop user service"`

- Install the updated Fleet package.
- Verify `/usr/lib/systemd/user/fleet-desktop.service` exists.
- Confirm `systemctl --global is-enabled fleet-desktop.service` shows enabled.
- Verify the symlink exists and resolves: `ls -la /opt/orbit/bin/desktop/fleet-desktop` should point to the versioned binary.

### 2. User Session Handling

**Note:** On first install, `systemctl --user status fleet-desktop.service` may show `inactive (dead)` with a stale failure from before orbit wrote the env file. This is expected — the `ConditionPathExists=/opt/orbit/desktop.env` directive prevents premature startup, but systemd caches the failed attempt. Once orbit starts and calls `restartServiceForLoggedInUsers()` (which does `daemon-reload` + `restart`), the service starts successfully. Verify by checking orbit logs for `"managing fleet-desktop as systemd user service"` and `"restarted fleet-desktop user service"`.

- Log in as a regular user with a graphical session (X11/Wayland).
- Confirm Fleet Desktop is running: `systemctl --user status fleet-desktop.service` should show `active (running)`.
- Log out and back in; verify the service stops/starts with the session.
- SSH in as a user (no GUI session) and verify `systemctl --user status fleet-desktop.service` shows inactive — service should NOT start for SSH-only users.

### 3. Environment Inheritance
- Check that Fleet Desktop inherits `DISPLAY`, `WAYLAND_DISPLAY`, and `DBUS_SESSION_BUS_ADDRESS` from the user session.
- Confirm `/opt/orbit/desktop.env` is readable and contains expected env vars.

### 4. Browser Launch Verification
- Trigger a browser launch from Fleet Desktop ("My device" link).
- Confirm the browser opens in the user's session (not as root, no permission errors).

### 5. Service Management
- Manually stop/start/restart the service:
  - `systemctl --user stop fleet-desktop.service`
  - `systemctl --user start fleet-desktop.service`
- Check `journalctl --user -u fleet-desktop` — confirm logs go to the user journal, not system journal.
- Verify orbit logs (`journalctl -u orbit`) show `"managing fleet-desktop as systemd user service"` confirming the new code path is active.

### 6. Edge Cases
- **Multiple users**: Log in as two users with graphical sessions; confirm each gets their own Fleet Desktop instance.
- **Token file**: Ensure `/opt/orbit/identifier` is `0644` and readable by users.
- **Orbit restart**: Run `systemctl restart orbit`; confirm the fleet-desktop user service gets restarted too.
- **Token refresh**: After orbit rotates the token, verify permissions stay `0644` (the periodic `ensureTokenReadable` check).
- **Remove package**: Uninstall Fleet; verify service is disabled and removed, and processes are stopped.
- **Upgrade**: Simulate upgrade from old package; confirm migration from old runner to user service.
- **Headless/server distro**: Install on a minimal server with no graphical target — confirm the service is enabled but never starts (no errors from systemd).

### 7. Negative/Failure Scenarios
- Remove `/opt/orbit/desktop.env`; restart service, confirm failure and error logs in `journalctl --user -u fleet-desktop`.
- Remove user's graphical session; confirm service stops.
- Kill the `fleet-desktop` process with `kill -9` — confirm systemd restarts it automatically (`Restart=on-failure`).
- Corrupt the symlink (`rm /opt/orbit/bin/desktop/fleet-desktop`) — confirm the service fails cleanly with a useful error in `journalctl --user`.

### Out of scope (PoC)
- TUF auto-updates of the desktop binary mid-session. The current flow requires an orbit restart to pick up a new binary, which is unchanged.
