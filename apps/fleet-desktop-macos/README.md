# Fleet Desktop (macOS)

A native macOS application that provides end users with a self-service portal for [Fleet](https://fleetdm.com). It integrates with Fleet's [orbit](https://fleetdm.com/docs/get-started/anatomy#orbit) agent to give users direct access to device management features in a native window instead of a browser.

> Fleet's agent already ships the "Fleet Desktop menu bar icon" (bundle ID `com.fleetdm.desktop`, built from `orbit/cmd/desktop`). This is the separate "Fleet Desktop app" (bundle ID `com.fleetdm.fleet-desktop`) distributed as its own `.pkg`. They use different bundle IDs and can coexist. [Learn more](https://fleetdm.com/guides/fleet-desktop).

## Features

- **Native macOS app** built with Swift and AppKit
- **Universal binary** supporting Apple Silicon (arm64) and Intel (x86_64)
- **Self-service portal** embedded in a native window via WKWebView
- **Automatic token refresh** handles hourly token rotation transparently
- **Loading screen** with Fleet logo while the portal loads
- **File download support** for `.mobileconfig` profiles and other files served by Fleet
- **Dark/light mode** respects the user's system appearance
- **`fleet://` URL scheme** for deep linking to Self-service, Policies, and triggering refetches
- **MDM required** — both the app and installer enforce MDM enrollment
- **Code signed and notarized** for secure distribution via `.pkg` installer

## Requirements

- macOS 13.0 (Ventura) or later
- MDM-enabled Mac with Fleet's managed preferences profile installed
- Fleet's orbit agent installed and enrolled
- The orbit identifier file must exist at `/opt/orbit/identifier`

## Installation

The signed, notarized `.pkg` is produced by CI (see [CI/CD](#cicd)) and uploaded as a workflow artifact. To deploy:

- **Via Fleet (Software):** upload the `.pkg` to Fleet as a software installer. Fleet Desktop will appear in the software catalog for deployment.
- **Manually:** double-click the `.pkg` and follow the installer.

The installer requires an MDM-enabled Mac. It checks for the Fleet managed preferences profile before proceeding — if the profile is not found, the installer displays an error and aborts. The app is placed in `/Applications` with `root:admin` ownership and `755` permissions. On upgrades, the installer gracefully quits Fleet Desktop before installing and automatically relaunches it afterward.

## How It Works

1. **Reads the Fleet URL** from MDM managed preferences (see [Configuration Sources](#configuration-sources))
2. **Reads the device token** from `/opt/orbit/identifier` (managed by orbit, rotates hourly)
3. **Opens the self-service portal** at `{FleetURL}/device/{token}/self-service` in an embedded browser window

### Token Rotation

The device token in `/opt/orbit/identifier` rotates every hour. Fleet Desktop handles this automatically:

- A background timer checks the identifier file every 60 seconds (and keeps the Dock badge current even when the window is closed)
- On HTTP 401/403 errors or error page detection, the app immediately checks for a new token and retries (up to 3 attempts with 5-second delays)
- Token refreshes are invisible to the user — the page silently reloads with the new token

### File Downloads

When Fleet serves downloadable content (e.g., MDM enrollment profiles):

- `.mobileconfig` files are downloaded and automatically opened for installation
- All other file types (`.pkg`, `.dmg`, `.zip`, etc.) are saved to `~/Downloads`

### Security

- App Transport Security (ATS) is enforced for the in-app WebView — the embedded portal requires HTTPS
- External links are restricted to `https`, `http`, and `mailto` schemes
- Device tokens are percent-encoded and not exposed in error messages
- Downloaded files are only auto-opened if they are `.mobileconfig` profiles
- The WebView uses a non-persistent data store (no cookies or cache persist between sessions)
- Mutable state is protected by a serial dispatch queue for thread safety

## Development

### Project Structure

```
apps/fleet-desktop-macos/
├── FleetDesktop/
│   ├── FleetDesktopApp.swift   # App delegate, main menu, entry point
│   ├── FleetService.swift      # Config reading, token management, refresh timer
│   ├── BrowserWindow.swift     # WKWebView window, loading overlay, downloads
│   ├── Info.plist              # App bundle metadata
│   ├── AppIcon.icns            # App icon
│   └── fleet-logo.png          # Fleet logo for loading screen
├── build.sh                    # Compiles universal binary
└── build-pkg.sh                # Creates the .pkg installer
```

The CI workflow lives at [`.github/workflows/fleet-desktop-macos-build.yml`](../../.github/workflows/fleet-desktop-macos-build.yml).

### Building Locally

```bash
# Build the app
./build.sh

# Run
open "build/Fleet Desktop.app"

# Build the (unsigned) .pkg installer
./build-pkg.sh
```

Local builds are unsigned. Signing and notarization happen in CI with Fleet's Developer ID certificates.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ORBIT_ROOT_DIR` | `/opt/orbit` | Override the orbit directory (changes where the identifier file is read from) |

### Configuration Sources

| File | Key | Purpose |
|------|-----|---------|
| `/Library/Managed Preferences/com.fleetdm.fleetd.config.plist` | `FleetURL` | Fleet server URL (delivered via MDM profile) |
| `/opt/orbit/identifier` | — | Device authentication token (rotates hourly) |

> **Note:** Fleet Desktop only supports MDM-enabled Macs. If the managed preferences file is not present, the app displays an error and the installer refuses to proceed.

### URL Scheme

Fleet Desktop registers the `fleet://` URL scheme, allowing other tools and scripts to open specific pages:

| URL | Action |
|-----|--------|
| `fleet://self-service` | Opens the Self-service tab |
| `fleet://software` | Opens the Software tab |
| `fleet://policies` | Opens the Policies tab |
| `fleet://refetch` | Triggers a device refetch and opens the app |
| `fleet://update_all` | Opens Self-service and clicks "Update all" |
| `fleet://anything-else` | Brings the app to the foreground |

Example usage from a script or terminal:

```bash
open fleet://self-service
open fleet://refetch
```

## CI/CD

[`.github/workflows/fleet-desktop-macos-build.yml`](../../.github/workflows/fleet-desktop-macos-build.yml) runs on pull requests touching `apps/fleet-desktop-macos/**`, on push to `main`, and via manual dispatch. It:

1. Compiles a universal binary (arm64 + x86_64)
2. Code signs the app with Fleet's Developer ID Application certificate
3. Packages into a `.pkg` installer with a custom distribution XML
4. Signs the `.pkg` with Fleet's Developer ID Installer certificate
5. Notarizes with Apple and staples the ticket
6. Uploads the signed `.pkg` as a workflow artifact (retained for 30 days)

Pull requests (including from forks) only run step 1 — they verify the app compiles and packages, but skip signing/notarization, which require secrets unavailable to forks.

### Signing secrets

The workflow reuses the same repository secrets already used by Fleet's other macOS build workflows — **no new secrets are required**:

| Secret | Purpose |
|--------|---------|
| `APPLE_APPLICATION_CERTIFICATE` / `..._PASSWORD` | Developer ID Application certificate (.p12, base64) + password |
| `APPLE_INSTALLER_CERTIFICATE` / `..._PASSWORD` | Developer ID Installer certificate (.p12, base64) + password |
| `APPLE_USERNAME` / `APPLE_PASSWORD` | Apple ID + app-specific password for notarization |
| `APPLE_TEAM_ID` | Apple Developer Team ID |
| `KEYCHAIN_PASSWORD` | Temporary CI keychain password |

The Developer ID certificate identities (SHA-1) are pinned in the workflow `env` block, matching the identities used by Fleet's orbit and fleetd-base builds.

## License

Licensed under the MIT Expat license via the repository [root LICENSE](../LICENSE).
