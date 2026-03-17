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
                      └─ systemctl --global enable fleet-desktop.service

systemd (user)   → fleet-desktop.service (per-user, auto-started on login)
                      └─ EnvironmentFile=/opt/orbit/desktop.env
                      └─ ExecStart=/opt/orbit/bin/orbit/desktop/linux/stable/fleet-desktop
```

## Key design decisions

### 1. Systemd user service unit file

Install to `/usr/lib/systemd/user/fleet-desktop.service` during packaging. This path is used by systemd for vendor-provided user units. Using `systemctl --global enable` makes it start for ALL users on login.

```ini
[Unit]
Description=Fleet Desktop
After=graphical-session.target
PartOf=graphical-session.target

[Service]
Type=simple
EnvironmentFile=/opt/orbit/desktop.env
ExecStart=/opt/orbit/bin/orbit/desktop/linux/stable/fleet-desktop
Restart=on-failure
RestartSec=5

[Install]
WantedBy=graphical-session.target
```

Key choices:
- **`graphical-session.target`**: ensures desktop only starts when a graphical session is active — no more manual GUI user detection
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

Orbit refreshes this file whenever configuration changes, then signals the user service to restart via `systemctl --user restart fleet-desktop.service` for logged-in users.

### 3. Token file permissions

The token file at `/opt/orbit/identifier` is currently only readable by root (written by orbit). We need to make it readable by the user running desktop. Change permissions to `0644` so the user service can read it.

### 4. Orbit changes (what gets removed / changed)

**In orbit's main loop (`orbit.go`):**
- On Linux, when `--fleet-desktop` is set, instead of creating a `desktopRunner`, orbit:
  1. Writes `/opt/orbit/desktop.env` with the required env vars
  2. Ensures the token file has proper permissions (0644)
  3. On first run or config change, signals `systemctl --user daemon-reload` + restart for logged-in users

**The `desktopRunner` stays** for macOS/Windows (no changes to those platforms). On Linux, it's replaced by a simpler `desktopServiceManager` that just manages the env file and service state.

### 5. Packaging changes (`linux_shared.go`)

- Add `fleet-desktop.service` to `/usr/lib/systemd/user/` in the package
- **postinstall**: `systemctl --global enable fleet-desktop.service` + start for logged-in users
- **preremove**: `systemctl --global disable fleet-desktop.service` + stop for logged-in users
- **postremove**: clean up `/usr/lib/systemd/user/fleet-desktop.service` and `/opt/orbit/desktop.env`

### 6. Migration from old to new (upgrade path)

When upgrading from a package that uses the old `desktopRunner` approach:
- The new orbit binary won't launch desktop via `execuser.Run` on Linux anymore
- The postinstall script enables the user service
- The old `pkill fleet-desktop` in preremove handles killing the old process
- On next login, systemd starts the user service automatically

### 7. Desktop binary symlink

Add a stable symlink for the service to reference:
`/opt/orbit/bin/desktop/fleet-desktop` → actual versioned binary

This avoids hardcoding channel/platform paths in the service unit. Orbit updates this symlink when it updates the desktop binary (already handles this for the orbit binary itself).

## Files to modify

| File | Change |
|------|--------|
| `orbit/cmd/orbit/orbit.go` | Add `desktopServiceManager` for Linux; skip `desktopRunner` on Linux |
| `orbit/pkg/packaging/linux_shared.go` | Add user service unit file, update post/pre scripts, add desktop symlink |
| `orbit/pkg/packaging/packaging.go` | (if needed) Add desktop symlink to Options |
| `orbit/cmd/orbit/desktopservice_linux.go` | New file: `desktopServiceManager` implementation |
| `orbit/cmd/orbit/desktopservice_other.go` | New file: build tag stub for non-Linux |

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

6. **TUF updates**: When orbit updates the desktop binary, it needs to restart the user service. This is done by running `systemctl --user restart fleet-desktop.service` for each logged-in user.
