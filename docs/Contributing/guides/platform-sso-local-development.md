# Local development: Apple Platform SSO (PSSO)

This guide walks a Fleet contributor through standing up a working Apple [Platform Single Sign-On](https://support.apple.com/guide/deployment/platform-single-sign-on-dep7bbb05313/web) (PSSO) dev environment end to end: a locally built, dev-signed Fleet PSSO extension talking to your local Fleet server, so you can exercise device registration, password login, and key exchange against a real Mac.

PSSO is a Fleet Premium feature and targets macOS 26+. The end-user feature is documented for admins in the Setup Experience guide; this guide is for engineers hacking on the implementation.

## Why local PSSO takes setup

PSSO has a three-way chain of trust that all has to agree on the same `<TeamID>.<BundleID>`:

1. **The signed extension.** `FleetPSSOExtension.appex` carries Apple-*managed* entitlements (`com.apple.developer.associated-domains`). `codesign` only honors those when a provisioning profile from a real Apple Developer team is embedded — an ad-hoc signature is not enough for PSSO to engage.
2. **The AASA document.** Your Fleet server must serve `/.well-known/apple-app-site-association` listing the exact `<TeamID>.<BundleID>` of the signed extension. If they don't match, associated-domains validation fails silently and PSSO never starts.
3. **The configuration profile.** The `com.apple.extensiblesso` + `com.apple.associated-domains` payloads must name the same extension bundle ID, team, and server host.

Because Apple App IDs are globally unique across teams, the production bundle IDs (`com.fleetdm.fleet-desktop*`) can only be registered under Fleet's production team. Local development therefore uses a **separate, non-production Fleet dev team** with its own App IDs. The server's published AASA IDs are made overridable so your dev server can advertise that team.

## Prerequisites

- A **test Mac** running macOS 26+ enrolled in your local Fleet.
- A Fleet Premium or dev license and Apple MDM configured on your local server.
- Xcode command-line tools (`swiftc`, `codesign`, `PlistBuddy`).
- A public HTTPS tunnel to your local server — [ngrok](https://ngrok.com) or [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/). Apple fetches the AASA over HTTPS from your server's hostname, so `localhost` alone won't do.
- An IdP that supports the OAuth Resource Owner Password (ROPG) grant (Okta or Entra — see the deployment articles).
- The **shared dev signing assets** (see Step 1).

## Step 1: Get the shared dev signing assets

The Fleet dev PSSO team's signing certificate and provisioning profiles are shared through 1Password (search for the *"Apple Fleet PSSO dev signing certs"* and *"Apple Fleet PSSO dev signing Provision Profiles"*) — they are **never** committed to the repo, the same convention CI uses for the production profiles. The item contains:

- A **`.p12`** holding the dev team's two Developer ID certificates **with their private keys** — import into your login keychain (double-click, or `security import`). These are the *dev team's* certs (team `5K28R5ZUK5`), **not** the production `Fleet Device Management Inc` certs CI uses:
  - **Developer ID Application** (SHA-1 `B5D91FADAD41D3DF1BF0CDD7A4EFADE73845FEA6`) — signs the `.app`/`.appex`. The provisioning profiles below authorize this one.
  - **Developer ID Installer** (SHA-1 `2412733379527DEC91D6E8376F5E65B7DDAA2D1E`) — signs the `.pkg`, so it can be pushed through Fleet / MDM like production.
- Two **Developer ID** provisioning profiles, for the dev App IDs:
  - `com.fleetdm.pssotesting` (host app) - PSSO_Testing_App.provisionprofile
  - `com.fleetdm.pssotesting.extension` (extension) - PSSO_Testing_Extension.provisionprofile

The profiles are registered under the dev team with the **Associated Domains** and **MDM Managed Associated Domains** capabilities enabled, and authorize only the Application cert above — AMFI kills the app at launch if you sign with any other (see the note in Step 2). Save the two `.provisionprofile` files somewhere outside the repo (e.g. `~/psso-dev/`).

Confirm both identities imported (the `.p12` carries the private keys the certs alone don't):

```bash
security find-identity -v -p basic | grep 5K28R5ZUK5
# ...  B5D91FADAD41D3DF1BF0CDD7A4EFADE73845FEA6 "Developer ID Application: ... (5K28R5ZUK5)"
# ...  2412733379527DEC91D6E8376F5E65B7DDAA2D1E "Developer ID Installer: ... (5K28R5ZUK5)"
```

> If you'd rather use your own Apple Developer team, you can — register your own two App IDs (with those two capabilities), Developer ID Application + Installer certs, and provisioning profiles, then substitute your team ID, bundle IDs, and signing certs throughout. The shared team just saves everyone that setup.

## Step 2: Build, sign, and package the extension under the dev team

You'll almost always test on a *different* Mac than you build on, PSSO misconfiguration can leave your machine in a very bad state, so the test machine is rarely your dev box. The flow is: build + sign the app on your dev machine, package it into a `.pkg`, then deploy that `.pkg` to the test Mac along with a profile configuring it.

### Build and sign

`build.sh` compiles the universal app + extension and, when given signing inputs, signs them inside-out with the dev team's identity, bundle IDs, and profiles (leaving them unset reproduces the compile-only bundle CI signs separately):

```bash
cd apps/fleet-desktop-macos

TEAM_ID=5K28R5ZUK5 \
APP_BUNDLE_ID=com.fleetdm.pssotesting \
EXT_BUNDLE_ID=com.fleetdm.pssotesting.extension \
SIGNING_IDENTITY=B5D91FADAD41D3DF1BF0CDD7A4EFADE73845FEA6 \
APP_PROFILE=~/psso-dev/PSSO_Testing_App.provisionprofile \
EXT_PROFILE=~/psso-dev/PSSO_Testing_Extension.provisionprofile \
./build.sh
```

(`SIGNING_IDENTITY` is the dev cert's SHA-1 from Step 1; `codesign` accepts either the hash or the full `"Developer ID Application: … (5K28R5ZUK5)"` name.)

### Package into a signed .pkg

Build the installer with the **same** `APP_BUNDLE_ID` (so the pkg and its embedded app agree) and the Installer cert, so the pkg is signed and can be pushed through Fleet / MDM like production:

```bash
INSTALLER_SIGNING_IDENTITY="2412733379527DEC91D6E8376F5E65B7DDAA2D1E" \
APP_BUNDLE_ID=com.fleetdm.pssotesting \
./build-pkg.sh
# → build/dist/fleet_desktop-v<version>.pkg  (signed)
```

`build-pkg.sh` reuses the already-signed app from the previous step (no rebuild unless `FORCE_REBUILD=1`) and `productsign`s the finished pkg. Find the Installer identity's name or SHA-1 with `security find-identity -v -p basic | grep 5K28R5ZUK5` (either form works for `--sign`). Confirm the result:

```bash
pkgutil --check-signature build/dist/fleet_desktop-v<version>.pkg
```

> Omitting `INSTALLER_SIGNING_IDENTITY` produces an **unsigned** pkg — fine for a manual `installer -pkg` on your own test Mac (see below), but Fleet/MDM pushes need the signature.

### Deploy to your test Mac

Pick whichever matches what you're exercising:

**Push through Fleet (mirrors production).** Upload the signed `.pkg` under **Software → Add software** as a custom package, then install it on the enrolled test host (from the host's **Software** tab, or by adding it to the team's install list). Fleet's agent downloads and installs it on the Mac. This is the path most contributors will use.

**Manual install (fastest for iterating).** Copy the pkg to the test Mac and install it directly:

```bash
# on the test Mac
sudo installer -pkg fleet_desktop-v<version>.pkg -target /
```

Either way, installing into `/Applications` is what registers the embedded `.appex` so a profile can select it. A locally-built pkg copied over `scp`/USB has no `com.apple.quarantine` flag and `installer` as root skips the Gatekeeper UI gate, so an unsigned pkg also installs cleanly for the manual path.

> If you're building **on** the test Mac itself, skip the pkg entirely and `sudo ditto "build/Fleet Desktop.app" "/Applications/Fleet Desktop.app"`.

> **Notarization.** Signing is enough for dev: Fleet's agent installs as root and the extension's validity comes from its embedded provisioning profile, not Gatekeeper. Only notarize (an extra step needing Apple ID credentials, not just the `.p12`) if you hit a Gatekeeper block — it's the same step CI runs, see [`README.md`](../../../apps/fleet-desktop-macos/README.md#cicd).

### Verify

On the test Mac, confirm the signature and that the system sees the extension:

```bash
codesign -dv --entitlements - "/Applications/Fleet Desktop.app/Contents/PlugIns/FleetPSSOExtension.appex"
pluginkit -m | grep pssotesting
```

> **App SIGKILLed at launch?** The provisioning profile must authorize the exact certificate you signed with, or AMFI kills the app (notarization/Gatekeeper won't catch this). Dump the profile with `security cms -D -i <profile>.provisionprofile` and confirm your signing cert appears in its `DeveloperCertificates`.

## Step 3: Expose your Fleet server over HTTPS

Start your local Fleet server (default `https://localhost:8080`) and point a tunnel at it:

```bash
cloudflared tunnel --url https://localhost:8080
# or: ngrok http https://localhost:8080
```

Note the public URL it prints (e.g. `https://myenv.ngrok.app`). Set your server's `server_url` (org settings, or `FLEET_SERVER_URL`) to that tunnel URL — the extension endpoints and the AASA are all derived from it.

## Step 4: Point the server's AASA at the dev team

Fleet's built-in AASA advertises the production team. Run the server with `--dev` and the override so it advertises your dev team's App IDs instead:

```bash
FLEET_DEV_PSSO_AASA_APP_IDS="5K28R5ZUK5.com.fleetdm.pssotesting,5K28R5ZUK5.com.fleetdm.pssotesting.extension" \
  ./build/fleet serve --dev   # ...plus your usual serve flags
```

The override is read through Fleet's dev-mode env mechanism, so it is **only** honored under `--dev`; a production server ignores it entirely and always publishes Fleet's own App IDs.

Confirm the document matches your signed binary (through the tunnel, so you hit the same host Apple will):

```bash
curl https://myenv.ngrok.app/.well-known/apple-app-site-association
# {"authsrv":{"apps":["5K28R5ZUK5.com.fleetdm.pssotesting","5K28R5ZUK5.com.fleetdm.pssotesting.extension"]}}
```

## Step 5: Configure the IdP

Set up the upstream IdP for the ROPG grant and enter its details in Fleet's account provisioning settings (`oauth_idp_token_url`, `oauth_idp_client_id`, `oauth_idp_client_secret`). Follow the steps in the Setup Experience guide.

> **Okta ROPG gotcha:** the custom authorization server's **Access Policies** (Security → API → Authorization Servers → default → Access Policies) must have a rule with **Resource Owner Password** enabled, and the app's authentication policy must be password-only — otherwise the token call returns `no_matching_policy` or `password_auth_denied_policy`.

## Step 6: Build and upload the configuration profile

Copy fleet-sso-extension-example.mobileconfig and change these values for your dev build (leave everything else, including `RegistrationToken`, alone — Fleet substitutes `$FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN` per host):

In the `com.apple.extensiblesso` payload:

| Key | Change to |
|-----|-----------|
| `ExtensionData` → `BaseURL` | your tunnel URL, e.g. `https://myenv.ngrok.app` |
| `ExtensionIdentifier` | `com.fleetdm.pssotesting.extension` |
| `TeamIdentifier` | `5K28R5ZUK5` |
| `URLs` (array entry) | your tunnel URL |

In the `com.apple.associated-domains` payload, for **both** `Configuration` entries:

| Key | Change to |
|-----|-----------|
| `ApplicationIdentifier` | `5K28R5ZUK5.com.fleetdm.pssotesting` and `5K28R5ZUK5.com.fleetdm.pssotesting.extension` respectively |
| `AssociatedDomains` (array entry) | `authsrv:myenv.ngrok.app?mode=developer` (host only, no scheme) |

Upload the edited profile to your test host's team as a custom configuration profile.

> **`?mode=developer` matters for iteration.** Apple's CDN caches the AASA for hours (6–24h), so a plain `authsrv:host` entry will keep validating a stale document while you're changing things. Appending `?mode=developer` makes the device fetch the AASA **directly** from your server instead of via the CDN. Keep it on for all dev testing.

## Step 7: Trigger and observe

The Setup Assistant registration path fires PSSO for a freshly-enrolled account. To re-trigger on an already-set-up Mac, inspect and reset PSSO state with the `app-sso` CLI (run `app-sso platform --help` for the exact flags on your macOS version) — e.g. to print the current platform SSO state and to force re-registration.

The Fleet PSSO extension has no custom log subsystem; PSSO activity surfaces under Apple's SSO subsystems and the `AppSSOAgent` process. Stream them live:

```bash
log stream --level debug \
  --predicate 'subsystem CONTAINS "AppSSO" OR subsystem CONTAINS "PlatformSSO" OR process == "AppSSOAgent"'
```

The login-window / unlock path runs before you can start a stream, so for those capture after the fact and open the archive in Console (filter on the same subsystems):

```bash
sudo log collect --last 10m --output ~/psso.logarchive
open ~/psso.logarchive
```

On the server side, watch the requests hit `/api/mdm/apple/psso/{nonce,registration,token}` and `/.well-known/apple-app-site-association`.

## Troubleshooting

- **AASA doesn't match / PSSO never engages** — re-run the `curl` from Step 4 and confirm the app IDs equal what you signed. Remember the CDN cache: use `?mode=developer` and re-install the profile.
- **App killed immediately at launch** — the provisioning profile doesn't authorize your signing certificate (see Step 2's note).
- **`no_matching_policy` at login** — IdP ROPG isn't enabled (see Step 5).
- **Never commit signing assets** — the `.p12`, `.provisionprofile` files, and any built `.pkg` stay out of git; they live in 1Password.

## Related

- Extension internals, entitlements, and the CI signing pipeline: [`apps/fleet-desktop-macos/README.md`](../../../apps/fleet-desktop-macos/README.md)
- Protocol / design decisions: [`docs/Contributing/research/mdm/psso.md`](../research/mdm/psso.md)
