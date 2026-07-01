# Fleet Desktop (macOS)

A native macOS application that provides end users with a self-service portal for [Fleet](https://fleetdm.com). It integrates with Fleet's [orbit](https://fleetdm.com/docs/get-started/anatomy#orbit) agent to give users direct access to device management features in a native window instead of a browser.

It also embeds the **Fleet Platform SSO (PSSO) extension** (`FleetPSSOExtension.appex`), which implements Apple's Platform Single Sign-On v2 + Password Mode so Fleet can create a Mac's local account and keep its password in sync with the user's IdP credentials. See [Platform SSO extension](#platform-sso-extension) below.

> **Heads up — two things named "Fleet Desktop":** Fleet's agent already ships a tray/menu-bar component called Fleet Desktop (bundle ID `com.fleetdm.desktop`, built from `orbit/cmd/desktop`). This is a separate, standalone native app (bundle ID `com.fleetdm.fleet-desktop`) distributed as its own `.pkg`. They use different bundle IDs and can coexist. [Learn more](https://fleetdm.com/guides/fleet-desktop).

## Features

- **Native macOS app** built with Swift and AppKit
- **Universal binary** supporting Apple Silicon (arm64) and Intel (x86_64)
- **Self-service portal** embedded in a native window via WKWebView
- **Embedded Platform SSO extension** for IdP-based local account creation and password sync
- **Automatic token refresh** handles hourly token rotation transparently
- **Loading screen** with Fleet logo while the portal loads
- **File download support** for `.mobileconfig` profiles and other files served by Fleet
- **Dark/light mode** respects the user's system appearance
- **`fleet://` URL scheme** for deep linking to Self-service, Policies, and triggering refetches
- **MDM required** — both the app and installer enforce MDM enrollment
- **Code signed and notarized** for secure distribution via `.pkg` installer

## Requirements

- macOS 13.0 (Ventura) or later for the app; the PSSO extension requires macOS 14.0+ (the password-sync feature targets macOS 26+)
- MDM-enabled Mac with Fleet's managed preferences profile installed
- Fleet's orbit agent installed and enrolled
- The orbit identifier file must exist at `/opt/orbit/identifier`

## Installation

The signed, notarized `.pkg` is produced by CI (see [CI/CD](#cicd)) and uploaded as a workflow artifact. To deploy:

- **Via Fleet (Software):** upload the `.pkg` to Fleet as a software installer. Fleet Desktop will appear in the software catalog for deployment.
- **Manually:** double-click the `.pkg` and follow the installer.

The installer requires an MDM-enabled Mac. It checks for the Fleet managed preferences profile before proceeding — if the profile is not found, the installer displays an error and aborts. The app is placed in `/Applications` with `root:admin` ownership and `755` permissions. On upgrades, the installer gracefully quits Fleet Desktop before installing and automatically relaunches it afterward.

Installing the app into `/Applications` is also what registers the bundled `FleetPSSOExtension.appex` with the system so it becomes selectable by a `com.apple.extensiblesso` configuration profile.

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

## Platform SSO extension

`FleetPSSOExtension.appex` is an Apple `com.apple.AppSSO.idp-extension` that implements Platform SSO v2 in Password Mode. The Fleet server provides the IdP endpoints; the extension registers the device's keys with Fleet and proxies password sign-in / key exchange through it.

The extension is bundled inside the app at `Fleet Desktop.app/Contents/PlugIns/FleetPSSOExtension.appex` and ships in the same `.pkg`.

### How it binds to a Fleet server

The extension derives all of its endpoints from a single `BaseURL` value supplied in the `ExtensionData` dictionary of a `com.apple.extensiblesso` configuration profile (see [`fleet-sso-extension-example.mobileconfig`](./fleet-sso-extension-example.mobileconfig)):

```xml
<key>BaseURL</key> <string>https://fleet.example.com</string>
```

From that it derives, under `/api/mdm/apple/psso/`:

- `POST /nonce`        — single-use nonces for token requests
- `POST /registration` — device key registration
- `POST /token`        — password login / key request / key exchange
- `GET  /jwks`         — Fleet's PSSO signing public key

The Fleet server also serves an Apple App Site Association file at `https://<hostname>/.well-known/apple-app-site-association` containing an `authsrv` entry naming the extension's `<TeamID>.<BundleID>` — i.e. `8VBZ3948LU.com.fleetdm.fleet-desktop.pssoextension`.

Because the same generic, CI-built extension must work against *any* Fleet server, the associated domain is **not** baked into the binary. Instead:

- The entitlement `com.apple.developer.associated-domains` ships as an **empty array**, with `com.apple.developer.associated-domains.mdm-managed` set to `true`.
- The actual `authsrv:` domain (the configured Fleet server) is delivered at runtime by an MDM **AssociatedDomains** payload targeting the extension's bundle ID.

### Entitlements

Both the host app and the extension carry restricted (Apple-managed) entitlements. These are not freely assertable — `codesign` only honors them when a Developer ID **provisioning profile** that grants them is embedded in the bundle (see [Signing secrets](#signing-secrets)).

| Bundle | Entitlement | Value |
|--------|-------------|-------|
| App + extension | `com.apple.developer.associated-domains` | empty array (must exist) |
| App + extension | `com.apple.developer.associated-domains.mdm-managed` | `true` |
| Extension only | `com.apple.security.app-sandbox` | `true` |
| Extension only | `com.apple.security.network.client` | `true` |

The host app is deliberately **not** sandboxed — it reads `/opt/orbit/identifier` and the managed-preferences plist outside any container. App extensions are always sandboxed.

## Development

### Project Structure

```
apps/fleet-desktop-macos/
├── FleetDesktop/
│   ├── FleetDesktopApp.swift        # App delegate, main menu, entry point
│   ├── FleetService.swift           # Config reading, token management, refresh timer
│   ├── BrowserWindow.swift          # WKWebView window, loading overlay, downloads
│   ├── Info.plist                   # App bundle metadata
│   ├── FleetDesktop.entitlements     # Host-app entitlements (managed associated domains)
│   ├── AppIcon.icns                 # App icon
│   └── fleet-logo.png               # Fleet logo for loading screen
├── FleetPSSOExtension/
│   ├── AuthenticationViewController.swift            # Principal class (SSO request handler)
│   ├── AuthenticationViewController+PSSO.swift        # Registration handler
│   ├── AuthenticationViewController+Shared.swift      # Payload / key-ID / config helpers
│   ├── AuthenticationViewController+Networking.swift  # URLSession against Fleet
│   ├── Info.plist                                     # appex metadata (NSExtension dict)
│   └── FleetPSSOExtension.entitlements                # Extension entitlements
├── fleet-sso-extension-example.mobileconfig          # Example com.apple.extensiblesso profile
├── build.sh                         # Compiles the universal app + appex
└── build-pkg.sh                     # Creates the .pkg installer
```

The CI workflow lives at [`.github/workflows/fleet-desktop-macos-build.yml`](../../.github/workflows/fleet-desktop-macos-build.yml).

The PSSO extension is built as a Foundation app extension with `swiftc`: there is no `main()`; the entry point is `NSExtensionMain` and the principal class is loaded from the appex `Info.plist`. `swiftc`'s `-module-name` must match the module prefix in `NSExtensionPrincipalClass` (`FleetPSSOExtension`).

### Building Locally

```bash
# Build the app (with the embedded extension)
./build.sh

# Run
open "build/Fleet Desktop.app"

# Build the (unsigned) .pkg installer
./build-pkg.sh
```

Local builds are unsigned. Signing and notarization happen in CI with Fleet's Developer ID certificates and provisioning profiles. You can ad-hoc sign locally to sanity-check the bundle layout (the restricted entitlements won't be honored without a real profile):

```bash
codesign --force --options runtime --sign - \
  --entitlements FleetPSSOExtension/FleetPSSOExtension.entitlements \
  "build/Fleet Desktop.app/Contents/PlugIns/FleetPSSOExtension.appex"
codesign --force --options runtime --sign - \
  --entitlements FleetDesktop/FleetDesktop.entitlements "build/Fleet Desktop.app"
codesign --verify --deep --strict "build/Fleet Desktop.app"
```

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

1. Compiles a universal binary (arm64 + x86_64) for the app and the extension, and assembles the `.appex` inside the `.app`
2. Embeds the Developer ID provisioning profiles into the app and extension bundles
3. Code signs **inside-out** — the extension first, then the host app — each with its own entitlements
4. Packages into a `.pkg` installer with a custom distribution XML
5. Signs the `.pkg` with Fleet's Developer ID Installer certificate
6. Notarizes with Apple and staples the ticket
7. Uploads the signed `.pkg` as a workflow artifact (retained for 30 days)

The workflow always signs and notarizes. Runs without access to the signing secrets — fork pull requests, or any run before the provisioning profiles have been added — **fail** rather than producing an unsigned artifact.

### Signing secrets

The workflow reuses the Developer ID certificate secrets already used by Fleet's other macOS build workflows, plus **two new provisioning-profile secrets** required for the extension's restricted entitlements:

| Secret | Purpose |
|--------|---------|
| `APPLE_APPLICATION_CERTIFICATE` / `..._PASSWORD` | Developer ID Application certificate (.p12, base64) + password |
| `APPLE_INSTALLER_CERTIFICATE` / `..._PASSWORD` | Developer ID Installer certificate (.p12, base64) + password |
| `APPLE_USERNAME` / `APPLE_PASSWORD` | Apple ID + app-specific password for notarization |
| `APPLE_TEAM_ID` | Apple Developer Team ID |
| `KEYCHAIN_PASSWORD` | Temporary CI keychain password |
| `APPLE_FLEET_DESKTOP_APP_PROFILE_B64` | base64 of the Developer ID provisioning profile for `com.fleetdm.fleet-desktop` |
| `APPLE_PSSO_EXT_PROFILE_B64` | base64 of the Developer ID provisioning profile for `com.fleetdm.fleet-desktop.pssoextension` |

The Developer ID certificate identities (SHA-1) are pinned in the workflow `env` block, matching the identities used by Fleet's orbit and fleetd-base builds.

#### Provisioning profiles (one-time Apple Developer portal setup)

The `com.apple.developer.associated-domains*` entitlements are Apple-managed: `codesign` will not honor them without a Developer ID provisioning profile that grants them. Profiles are **not committed** — they are team/cert-bound build inputs that expire, so they're stored as the base64 secrets above (the same pattern as the `.p12` certs).

Under Fleet's Apple Developer team (`8VBZ3948LU`, the team that owns the pinned Developer ID certificates):

1. Register two App IDs:
   - `com.fleetdm.fleet-desktop` (host app)
   - `com.fleetdm.fleet-desktop.pssoextension` (extension)
2. Enable the **Associated Domains** and **MDM Managed Associated Domains** capabilities on both App IDs.
3. Create a **Developer ID** provisioning profile (distribution, platform macOS) for each App ID. **Select the same Developer ID Application certificate that CI signs with** — SHA-1 `604D877399AAEB7630A78B84F288E2D28A2EDE42` (the identity pinned in the workflow). Fleet has more than one "Developer ID Application: Fleet Device Management Inc" certificate; a profile generated against the wrong one will sign and **notarize successfully but get SIGKILLed by AMFI at launch**, because AMFI requires the signing cert to appear in the profile's `DeveloperCertificates`. The `Verify profiles authorize the signing certificate` workflow step guards against this.
4. base64-encode each downloaded `.provisionprofile` and store them as `APPLE_FLEET_DESKTOP_APP_PROFILE_B64` and `APPLE_PSSO_EXT_PROFILE_B64`:
   ```bash
   base64 -i FleetDesktop_DeveloperID.provisionprofile | pbcopy   # → APPLE_FLEET_DESKTOP_APP_PROFILE_B64
   base64 -i FleetPSSOExtension_DeveloperID.provisionprofile | pbcopy  # → APPLE_PSSO_EXT_PROFILE_B64
   ```

Re-encode and update the secrets when a profile expires or the signing certificate is rotated. To inspect a profile — its entitlements and, crucially, the certs it authorizes — dump it with `security cms -D -i <profile>.provisionprofile`; the `DeveloperCertificates` array must contain the CI signing cert above.

## License

Licensed under the MIT Expat license via the repository [root LICENSE](../LICENSE).
