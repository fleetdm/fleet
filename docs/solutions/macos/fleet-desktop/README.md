# Fleet Desktop

Fleet Desktop is a native macOS application that provides end users with a self-service portal for [Fleet](https://fleetdm.com).

## Features

- **Native macOS app** built with Swift and AppKit
- **Universal binary** supporting Apple Silicon (arm64) and Intel (x86_64)
- **Self-service portal** embedded in a native window via WKWebView
- **Automatic token refresh** handles hourly token rotation transparently
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

### From Releases

1. Download the latest `fleet_desktop-v*.pkg` from the [Releases](https://github.com/allenhouchins/fleet-desktop/releases) page
2. Double-click the `.pkg` file to run the installer
3. Follow the installation wizard

The installer requires an MDM-enabled Mac. It checks for the Fleet managed preferences profile before proceeding — if the profile is not found, the installer will display an error and abort. The app is placed in `/Applications` with `root:admin` ownership and `755` permissions. On upgrades, the installer gracefully quits Fleet Desktop before installing and automatically relaunches it afterward.

### Via Fleet (Software)

Upload the `.pkg` to Fleet as a software installer. Fleet Desktop will appear in the software library for deployment.

## How It Works

1. **Reads the Fleet URL** from MDM managed preferences (see [Configuration Sources](#configuration-sources))
2. **Reads the device token** from `/opt/orbit/identifier` (managed by orbit, rotates hourly)
3. **Opens the self-service portal** at `{FleetURL}/device/{token}/self-service` in an embedded browser window

### Token Rotation

The device token in `/opt/orbit/identifier` rotates every hour. Fleet Desktop handles this automatically:

- A background timer checks the identifier file every 60 seconds (paused when the window is closed)
- On HTTP 401/403 errors or error page detection, the app immediately checks for a new token and retries (up to 3 attempts with 5-second delays)
- Token refreshes are invisible to the user — the page silently reloads with the new token

### Security

- App Transport Security (ATS) is enforced — only HTTPS connections are allowed
- External links are restricted to `https`, `http`, and `mailto` schemes
- Device tokens are percent-encoded and not exposed in error messages
- Downloaded files are only auto-opened if they are `.mobileconfig` profiles
- The WebView uses a non-persistent data store (no cookies or cache persist between sessions)
- Mutable state is protected by a serial dispatch queue for thread safety


### URL Scheme

Fleet Desktop registers the `fleet://` URL scheme, allowing other tools and scripts to open specific pages:

| URL | Action |
|-----|--------|
| `fleet://self-service` | Opens the Self-service tab |
| `fleet://software` | Opens the Software tab |
| `fleet://policies` | Opens the Policies tab |
| `fleet://refetch` | Triggers a device refetch and opens the app |
| `fleet://anything-else` | Brings the app to the foreground |

Example usage from a script or terminal:

```bash
open fleet://self-service
open fleet://refetch
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Support

- [Open an issue](https://github.com/fleetdm/fleet-desktop/issues) on GitHub
- [Fleet documentation](https://fleetdm.com/docs)
