# Testing Fleet Desktop updates menu locally

This guide walks through testing the Fleet Desktop **Updates** menu (self-service software in the tray) on your machine.

## Prerequisites

- **macOS** (Fleet Desktop runs on Windows/Linux too; paths below are macOS-oriented).
- Fleet server running locally (see [Building Fleet](building-fleet.md)).
- A host enrolled with Fleet that has **Fleet Desktop** (so a device token exists).

## 1. Run Fleet server

```sh
# From repo root
docker compose up -d
./build/fleet prepare db --dev
make generate-dev   # if you need the UI
./build/fleet serve --dev
```

Fleet UI: **https://localhost:8080**

## 2. Enable self-service and add software

The Updates section only appears when:

- **Software inventory** is enabled (org or team).
- The host has **self-service software** available (installers or VPP apps with `self_service: true` that are scoped to the host).

In Fleet UI:

1. **Settings → Organization settings** (or your team): ensure **Software inventory** is enabled.
2. **Software → Add software** (or use an existing title): add a macOS installer, enable **Self-service**, and set **Labels** so your test host is in scope (or use “No team” and no labels for a host in No team).

## 3. Get a host enrolled with Fleet Desktop

You need one enrolled host so the device token file exists. Either:

**Option A – Install from a locally built pkg (recommended for testing your code)**

See [Run locally built Fleetd](run-locally-built-fleetd.md). Use:

```sh
SYSTEMS="macos" \
PKG_FLEET_URL=https://localhost:8080 \
PKG_TUF_URL=http://localhost:8081 \
GENERATE_PKG=1 \
ENROLL_SECRET=<your-enroll-secret> \
FLEET_DESKTOP=1 \
USE_FLEET_SERVER_CERTIFICATE=1 \
./tools/tuf/test/main.sh
```

Then install the generated `.pkg` on your Mac. That installs orbit + Fleet Desktop and creates the token file.

**Option B – Use an existing enrolled host**

If you already have a Mac enrolled with Fleet Desktop (e.g. from a previous install), you can use that host and its token path (see step 4).

## 4. Run your local Fleet Desktop binary

After at least one host is enrolled, the device token lives at:

- **macOS (pkg install):** `/opt/orbit/identifier`
- **Linux:** `/opt/orbit/identifier`
- **Windows:** under `C:\Program Files\Orbit` (see orbit’s `--root-dir`).

Build and run Fleet Desktop with the same URL and token path:

```sh
# From repo root
go build -o build/fleet-desktop ./orbit/cmd/desktop/

# macOS (skip TLS verify for localhost)
FLEET_DESKTOP_DEVICE_IDENTIFIER_PATH=/opt/orbit/identifier \
FLEET_DESKTOP_FLEET_URL=https://localhost:8080 \
FLEET_DESKTOP_INSECURE=1 \
./build/fleet-desktop
```

If your orbit root is different (e.g. you ran orbit with `--root-dir /tmp/orbit`), use that path:

```sh
FLEET_DESKTOP_DEVICE_IDENTIFIER_PATH=/tmp/orbit/identifier \
FLEET_DESKTOP_FLEET_URL=https://localhost:8080 \
FLEET_DESKTOP_INSECURE=1 \
./build/fleet-desktop
```

- You should see the Fleet icon in the menu bar.
- Open the menu: **Fleet Desktop v…**, **My device**, **Self-service**, and optionally **Updates (N)** with app items and **Install all**.

## 5. When the Updates section appears

- **Updates** shows only if the server reports self-service available (`/device/{token}/desktop` has `self_service: true`) **and** `/device/{token}/software?self_service=1` returns at least one title.
- The menu is refreshed on the same interval as the desktop summary (about every 5 minutes), or when you trigger a refresh (e.g. open “My device” or “Self-service” in the menu).
- Clicking an app line triggers install for that title; **Install all** opens the self-service page in the browser.

## 6. Troubleshooting

- **No Updates section:** Confirm the host has software inventory enabled and at least one self-service installer/VPP app in scope for that host (team/labels). Check **My device** in the browser (same token) and see if the self-service list shows titles there.
- **“missing URL environment FLEET_DESKTOP_DEVICE_IDENTIFIER_PATH”:** Pass `FLEET_DESKTOP_DEVICE_IDENTIFIER_PATH` to the path of the `identifier` file (see step 4).
- **Connection/auth errors:** Use `FLEET_DESKTOP_INSECURE=1` for localhost HTTPS. If you use a custom CA, set `FLEET_DESKTOP_FLEET_ROOT_CA` to the CA cert path.
- **Desktop logs (macOS):** `~/Library/Logs/Fleet/fleet-desktop.log`
