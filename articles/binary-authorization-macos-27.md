# Binary authorization on macOS 27: allow and deny lists with Fleet
**ROUGH DRAFT

Application allowlisting on the Mac has always meant buying something extra or deploying Santa. Gatekeeper checks whether code is signed and notarized, not whether IT approved it something something TODO

macOS 27 changes that. A new declarative configuration, `com.apple.configuration.app.settings`, uses the Endpoint Security framework to decide which binaries are allowed to execute. It covers standalone binaries and binaries embedded in app bundles, so a script the user pastes into Terminal is subject to the same policy as a double-clicked app.

The same declaration type also handles app launch rules on iOS, iPadOS, tvOS, and visionOS, and privacy permission defaults that replace PPPC. This guide covers the macOS binary keys.

## Requirements

- macOS 27 or later
- The Mac must be **supervised**.
- The binary keys are **system scope**. (`Privacy` on the same declaration is user scope on macOS. Fleet's support for user-scoped declarations is still in progress, so plan to ship the binary policy as a separate device-scoped declaration.)
- Nothing needs to be installed on the endpoint. This is native OS enforcement, not an agent.

## The keys

Everything lives under `Payload.Allowed`.

| Key | What it does |
|---|---|
| `AllowedBinaries` | If present, **only** binaries matching an entry can run. |
| `DeniedBinaries`  | Binaries matching an entry **can't** run. Everything else can. |
| `AlwaysAllowManagedApps`  | Boolean, default `false`. If `true`, managed apps are implicitly added to the effective allow list whenever `AllowedApps` or `AllowedBinaries` is present. |
| `AllowedApps` / `DeniedApps` | Bundle-ID allow/deny lists. Not macOS. |

Three behaviors worth noting:

1. **Deny wins.** If a binary matches both lists, it's blocked.
2. **The user sees an alert.** A blocked launch isn't silent — macOS tells the user. (Would love to see Apple improve this)
3. **Most of the OS is exempt.** Binaries in the signed, sealed system volume are always permitted, and Apple's docs state the device always runs system-critical processes. You do not need to enumerate `/usr/bin`.

`AlwaysAllowManagedApps` is the key that makes this maintainable. Apps deployed via `com.apple.configuration.app.managed`, or via `InstallApplication` with `InstallAsManaged` set to `TRUE`, are covered automatically — so anything Fleet installs as software is implicitly trusted and doesn't need a hand-written rule.

## How a binary is matched

Each entry in `AllowedBinaries` / `DeniedBinaries` is a dictionary of identifier fields. **A binary matches only when every field in the entry matches.** More fields means a narrower rule.

| Field | Type | Notes |
|---|---|---|
| `CDHash` | string | Code directory hash. Pins one exact build. |
| `TeamID` | string | Code signing team identifier. Use the literal `APPLE` (not an empty string) for Apple binaries with an empty team ID. |
| `SigningID` | string | Code signature signing identifier. |
| `PathPrefix` | string | Filesystem path prefix. |
| `SigningState` | string | One of `All` (default), `TestFlight`, `DeveloperID`, `Enterprise`, `AppStore`, `Apple`. |

Apple's schema spells out which combinations are legal, and they differ between the two lists:

**`AllowedBinaries`**
- Either `CDHash` or `TeamID` must be present.
- `SigningID`, `PathPrefix`, or `SigningState` may be present.

**`DeniedBinaries`**
- Either `CDHash`, `TeamID`, or `SigningID` must be present.
- `PathPrefix` or `SigningState` may be present.

The asymmetry matters: you can deny by `SigningID` alone, but you can't allow by `SigningID` alone — an allow rule always needs a hash or a team.

That gives you a tunable trust radius:

- **`TeamID` only** — trust everything a developer signs. Broad, low maintenance, survives updates.
- **`TeamID` + `SigningID`** — trust one product from that developer.
- **`CDHash`** — pin one exact build. Nothing else passes, including the next version of the same app.
- **`+ PathPrefix`** — additionally require the binary live in an expected location.

TODO Harry: CDHash` pinning breaks on every app update, and CD hashes are per-architecture. Unless you have a compliance requirement that demands it, `TeamID` (optionally plus `SigningID`) is the sane default. Worth testing what a universal binary reports and whether both slices need entries.

## Build the inventory before you write the policy

On macOS, `codesign` gives you everything the schema wants:

```sh
codesign -dvvv /Applications/Slack.app
```

Read `TeamIdentifier` for `TeamID`, `Identifier` for `SigningID`, and `CDHash` for `CDHash`. Apple's documentation points at this same command.

Across a fleet, query it instead. Fleet already collects code signing data through osquery:

```sql
SELECT
  apps.name,
  apps.bundle_identifier,
  signature.team_identifier,
  signature.identifier AS signing_id,
  signature.cdhash,
  signature.authority
FROM apps
JOIN signature ON signature.path = apps.path;
```

Run that as a query across the fleet, and the result set is a draft allow list — every signing identity actually present on your Macs, ranked by how many hosts have it. Anything with a team ID you don't recognize is worth a conversation before you turn enforcement on.

TODO TODO  - Apps are the easy half. This query won't see standalone CLI binaries in `/usr/local/bin`, `/opt/homebrew`, `~/bin`, or a developer's project directories, and those are exactly what the new controls reach that the old ones didn't. Need a second approach here like a `file`-table sweep of the usual directories joined to `signature`, or process-level telemetry. Worth writing and testing before publish.

## Example: deny list first

Start here. A deny list changes the least and can't lock anyone out of their own machine.

```json
{
  "Type": "com.apple.configuration.app.settings",
  "Identifier": "com.example.binaries.deny",
  "Payload": {
    "Allowed": {
      "DeniedBinaries": [
        {
          "TeamID": "ABCDE12345"
        },
        {
          "SigningID": "com.example.unsanctioned-tool"
        },
        {
          "TeamID": "FGHIJ67890",
          "SigningID": "com.vendor.legacy-agent"
        }
      ]
    }
  }
}
```

## Example: allow list

```json
{
  "Type": "com.apple.configuration.app.settings",
  "Identifier": "com.example.binaries.allow",
  "Payload": {
    "Allowed": {
      "AlwaysAllowManagedApps": true,
      "AllowedBinaries": [
        {
          "TeamID": "APPLE"
        },
        {
          "TeamID": "EQHXZ8M8AV",
          "SigningID": "com.google.Chrome"
        },
        {
          "TeamID": "JQ525L2MZD"
        },
        {
          "CDHash": "e1b2c3d4e5f60718293a4b5c6d7e8f9012345678",
          "PathPrefix": "/usr/local/bin/"
        }
      ]
    }
  }
}
```

With `AlwaysAllowManagedApps` set to `true`, everything Fleet deploys as software is already covered — the array only needs the things you *didn't* deploy.

TODO Verify the `APPLE` sentinel behaves as expected, and confirm whether an `AllowedBinaries` entry with `SigningState: "Apple"` is a cleaner way to express the same intent. Also verify whether an empty `AllowedBinaries` array is treated as "allow nothing" 

## Deliver it with Fleet

The declaration type only exists on macOS 27. Older Macs will reject it and surface an error in OS settings:

```
Error.UnknownDeclarationType: Unknown Declaration Type map[UnknownDeclarationType:com.apple.configuration.app.settings]
```

So scope it. In Fleet, go to **Labels**, add a dynamic label named `macOS 27+`, and use:

```sql
SELECT 1 FROM os_version WHERE major >= 27;
```


## Rolling it out without breaking everyone

There is **no audit-only mode** in the schema. `AllowedBinaries` enforces the moment it lands. Some coverage of WWDC26 suggests turning on "reporting mode" first — as far as Apple's published schema goes, that mode doesn't exist. Sequence the rollout yourself.

Suggested order:

1. **Inventory.** Run the signing queries above across the whole fleet. Let them run long enough to catch the once-a-quarter tools.
2. **Deny list in production.** Block the specific things you already know you don't want. Low risk, immediate value, and it exercises the delivery path.
3. **Allow list on a pilot label.** Ten volunteers, `AlwaysAllowManagedApps: true`, team IDs from the inventory. Watch for a week.
4. **Widen by role.** Engineering will have the longest tail by a wide margin. Expect to iterate there far more than on a standard laptop build.
5. **Document the exception path.** When someone gets blocked, they need a route that ends in a pull request, not a Slack DM to whoever is awake.

## Gotchas

- **`com.apple.applicationaccess.new` is deprecated in macOS 27.** If you're using the old app-launch-restriction profile, it's on the clock. Migrate to `com.apple.configuration.app.settings`.
- **Supervised only.** BYOD and user-enrolled Macs can't receive this.
- **Deny wins over allow.**
- **Blocked launches are visible to the user.** Plan the support conversation.
- **`Privacy` on the same declaration type is user scope on macOS**, and PPPC keys in `com.apple.TCC.configuration-profile-policy` are deprecated in macOS 27. Accessibility management via TCC profile is already *removed* in macOS 27 — that one is not a deprecation you can sit on. Separate guide.

TODO
> - Is there any status report or log channel that surfaces *what got blocked*? Without that, troubleshooting is guesswork. Check `com.apple.status.*` items and the Endpoint Security subsystem in unified logging.
> - What exactly happens to a running process when the policy tightens under it?
> - How does this interact with an existing third-party EDR that uses Endpoint Security?
> - Does `PathPrefix` match the binary's own path or the enclosing bundle's?
> - Confirm behavior for scripts: an unsigned shell script invoked via an allowed interpreter — allowed or blocked?

## Further reading

- [WWDC26 app management updates — Apple Platform Deployment](https://support.apple.com/guide/deployment/app-management-updates-depd567c9ffa/web)
- [AppSettings — Apple Developer Documentation](https://developer.apple.com/documentation/devicemanagement/appsettings)
- [AppSettingsAllowedObject — Apple Developer Documentation](https://developer.apple.com/documentation/devicemanagement/appsettingsallowedobject)
- [AppSettingsAllowed_BinaryIdentifierObject — Apple Developer Documentation](https://developer.apple.com/documentation/devicemanagement/appsettingsallowed_binaryidentifierobject)
- [What's new in managing Apple devices — WWDC26 session 206](https://developer.apple.com/videos/play/wwdc2026/206/)
- [apple/device-management](https://github.com/apple/device-management)
- [Configuration profiles — Fleet](https://fleetdm.com/guides/custom-os-settings)
- [Declarative device management: a primer — Fleet](https://fleetdm.com/articles/declarative-device-management-a-primer)

<meta name="category" value="guides">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="publishedOn" value="2026-08-05">
<meta name="articleTitle" value="Binary authorization on macOS 27: allow and deny lists with Fleet">
<meta name="description" value="TODO!">
