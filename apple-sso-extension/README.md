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
│   ├── Info.plist
│   └── FleetPSSOExtension.entitlements
└── README.md
```

The host app exists solely so the extension bundle is installable —
launching it once is enough for macOS to discover the bundled
`.appex`. After that the host app does nothing.

## How it fits with Fleet's server

The Fleet Go server exposes (paths are illustrative — confirm against
the actual handler registrations):

- `IssuerHostname`           — issuer string returned in JWTs
- `NonceEndpoint`            — single-use nonces for PSSO requests
- `JwksEndpoint`             — IdP public keys
- `TokenEndpoint`            — password-grant token exchange
- `RegistrationEndpoint`     — device registration callback

The extension picks these up from `loginManager.extensionData`, i.e.
the *second* arbitrary dictionary in the extensible-SSO profile.

## Configuration profile

Install a `com.apple.extensiblesso` profile referencing the extension
bundle ID and Team ID, with an `ExtensionData` dict that includes:

```xml
<key>IssuerHostname</key>      <string>fleet.example.com</string>
<key>NonceEndpoint</key>       <string>https://fleet.example.com/api/v1/fleet/psso/nonce</string>
<key>JwksEndpoint</key>        <string>https://fleet.example.com/api/v1/fleet/psso/jwks</string>
<key>TokenEndpoint</key>       <string>https://fleet.example.com/api/v1/fleet/psso/token</string>
<key>RegistrationEndpoint</key><string>https://fleet.example.com/api/v1/fleet/psso/register</string>
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

## Build / sign / notarize

```bash
# Build
xcodebuild -project "Fleet PSSO.xcodeproj" -scheme FleetPSSO \
  -configuration Release -derivedDataPath ./build clean build

# Sign (Developer ID Application certificate from your Apple Developer account)
codesign --force --options runtime --timestamp \
  --sign "Developer ID Application: Your Name (TEAMID)" \
  --entitlements FleetPSSOExtension/FleetPSSOExtension.entitlements \
  ./build/Build/Products/Release/FleetPSSO.app/Contents/PlugIns/FleetPSSOExtension.appex

codesign --force --options runtime --timestamp \
  --sign "Developer ID Application: Your Name (TEAMID)" \
  --entitlements FleetPSSO/FleetPSSO.entitlements \
  ./build/Build/Products/Release/FleetPSSO.app

# Notarize
ditto -c -k --keepParent ./build/Build/Products/Release/FleetPSSO.app FleetPSSO.zip
xcrun notarytool submit FleetPSSO.zip \
  --apple-id you@example.com --team-id TEAMID --password "@keychain:notary" --wait
xcrun stapler staple ./build/Build/Products/Release/FleetPSSO.app
```

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
