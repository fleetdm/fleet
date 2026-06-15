# Fleet PSSO (Platform Single Sign-On) Extension — POC

This is the macOS-side scaffolding for Fleet's Platform Single Sign-On
v2 + Password Mode proof of concept. The Fleet server provides the
IdP endpoints (nonce, JWKS, token, registration); this Xcode project
is what gets installed on a Mac, signed with a Developer ID, notarized,
and bound to Fleet via a `com.apple.extensiblesso` configuration profile.

> Status: POC scaffolding. The framework conformances compile and the
> registration request flow is wired end-to-end, but the password
> sign-in path is intentionally stubbed. Production hardening (token
> caching, error UX, retries, telemetry) is out of scope here.

## Layout

```
apple-sso-extension/
├── Fleet PSSO.xcodeproj/
├── FleetPSSO/                # Cocoa host app (empty shell)
│   ├── AppDelegate.swift
│   ├── Info.plist
│   └── FleetPSSO.entitlements
├── FleetPSSOExtension/       # The actual SSO extension
│   ├── AuthenticationViewController.swift
│   ├── AuthenticationViewController+PSSO.swift
│   ├── AuthenticationViewController+Shared.swift
│   ├── AuthenticationViewController+Networking.swift
│   ├── Info.plist
│   └── FleetPSSOExtension.entitlements
└── README.md
```

The host app exists solely so the extension bundle is installable —
launching it once is enough for macOS to discover the bundled
`.appex`. After that the host app does nothing.

## How it fits with Fleet's server

The Fleet server exposes the Platform SSO endpoints under
`/api/mdm/apple/psso/`:

- `POST /api/mdm/apple/psso/nonce`        — single-use nonces for token requests
- `POST /api/mdm/apple/psso/registration` — device key registration
- `POST /api/mdm/apple/psso/token`        — password login / key request / key exchange
- `GET  /api/mdm/apple/psso/jwks`         — Fleet's PSSO signing public key

The extension derives all of them from the single `BaseURL` value in
`loginManager.extensionData`, i.e. the arbitrary dictionary in the
extensible-SSO profile. The issuer/audience is the BaseURL's bare
hostname.

## Configuration profile

Install a `com.apple.extensiblesso` profile referencing the extension
bundle ID and Team ID, with an `ExtensionData` dict that includes only
the Fleet server URL (see `fleet-sso-extension-example.mobileconfig`
for a complete profile):

```xml
<key>BaseURL</key> <string>https://fleet.example.com</string>
```

The hostname must also be served as an Apple App Site Association
file at:

```
https://<hostname>/.well-known/apple-app-site-association
```

containing an `authsrv` entry that names the extension bundle's
`<TeamID>.<BundleID>`.

## Placeholders to fill in

| Placeholder           | Where                                          |
|-----------------------|------------------------------------------------|
| `fleet.example.com`   | both `.entitlements` files; AASA hosting      |
| `com.fleetdm.psso`    | `Fleet PSSO.xcodeproj/project.pbxproj`        |
| `com.fleetdm.psso.extension` | same                                    |
| Development Team ID   | Xcode → target → Signing & Capabilities       |

## Build / sign / package / notarize

`build.sh` runs the whole pipeline end-to-end:

```bash
# Apple Developer credentials for notarytool. AC_PASSWORD must be an
# app-specific password — use @keychain:<item> if you've stored one.
export AC_USERNAME="you@example.com"
export AC_TEAM_ID="TEAMID"
export AC_PASSWORD="@keychain:notary"

./build.sh
```

This produces `./FleetPSSO.pkg`, a Developer ID-signed and notarized
installer. The pkg drops `FleetPSSO.app` into `/Applications` (which also
registers the bundled `.appex` with the system).

Install it:

```bash
sudo installer -pkg FleetPSSO.pkg -target /
```

Behind the scenes, the script:

1. Builds the app unsigned with `xcodebuild`.
2. Signs the `.appex` and `.app` with **Developer ID Application** (the
   hardened runtime + secure timestamp options that notarization
   requires).
3. Wraps the `.app` in a flat installer with `pkgbuild`, signs it with
   **Developer ID Installer**, and sets the install location to
   `/Applications`.
4. Submits the pkg to `notarytool` and waits for the verdict.
5. Staples the notarization ticket to the pkg so it installs offline.

You'll need both certificates in your login keychain:
- *Developer ID Application: Your Name (TEAMID)*
- *Developer ID Installer: Your Name (TEAMID)*

## Out of scope (intentional)

- Real password sign-in UI / token caching
- Keychain persistence (the framework owns key material via
  `ASAuthorizationProviderExtensionLoginManager`)
- Refresh, revocation, multi-account
- Pretty error / progress UX

## Open Apple API questions for the implementer

- `ASAuthorizationProviderExtensionLoginManager.userDeviceKey(forKeyType:)`
  was the throwing variant on macOS 14; double-check the signature on
  the macOS SDK you build against — Apple has both throwing and
  completion-handler variants depending on release.
- `ASAuthorizationProviderExtensionLoginConfiguration.supportedGrantTypes`
  on the password path: confirm whether `.password` is the correct
  case name on your SDK.
- AASA `authsrv:` entry format vs. `webcredentials:` — for PSSO it is
  `authsrv:` per WWDC 2022.

## Debug notes

authsrv: links and swcutil/swcd were the biggest stumbling block I ran into
