# Binary authorization on macOS 27: allow and deny lists with native Apple features

Binary allowlisting on the Mac has, in practice, meant buying something extra or deploying Santa. macOS has plenty of built-in application security. Gatekeeper, for example, checks whether code is signed and notarized, but not whether IT approved it to run in your environment. macOS 27 adds binary allow and deny lists to a new declarative configuration, `com.apple.configuration.app.settings`. Enforcement runs on the Endpoint Security framework and decides which binaries are allowed to execute. It covers standalone binaries as well as binaries embedded in app bundles. A tool a user downloads and runs from Terminal is subject to the same policy as a double-clicked app.

The same declaration exists on iOS, iPadOS, tvOS, and visionOS, but there it controls apps by bundle ID (`AllowedApps` / `DeniedApps`). The binary keys are macOS-only, and that's what this guide covers.

## Requirements

- macOS 27 or later (which also means Apple silicon; macOS 27 doesn't support Intel Macs).
- The Mac must be supervised, which on macOS means enrolled through Automated Device Enrollment. Personally owned and User Enrollment Macs can't receive this declaration.
- The binary keys are system scope, so deliver them as a device-scoped declaration.
- Nothing needs to be installed on the device. This is native OS enforcement, not an agent.

> **Note:** `com.apple.applicationaccess.new` is deprecated in macOS 27. If you're using the old app launch restriction profile, it's on the clock. Migrate to `com.apple.configuration.app.settings`.

## Declaration keys

Everything lives under `Payload.Allowed`.

| Key | What it does |
|---|---|
| `AllowedBinaries` | If present, only binaries matching an entry can run. |
| `DeniedBinaries` | Binaries matching an entry can't run. Everything else can. |
| `AlwaysAllowManagedApps` | Boolean, default `false`. If `true`, managed apps are implicitly added to the effective allow list whenever `AllowedBinaries` is present. |

`AlwaysAllowManagedApps` is the key that makes an allow list maintainable. Apps deployed with `com.apple.configuration.app.managed`, or with the `InstallApplication` command and `InstallAsManaged` set to `true`, are covered automatically.

> **Note:** Deny wins. You can use both lists in one declaration. When you do, the device runs only what the allow list permits, minus anything the deny list blocks. If a binary appears in both, it's blocked.

## How a binary is matched

Each entry in `AllowedBinaries` or `DeniedBinaries` is a dictionary of identifier fields. A binary matches only when every field in the entry matches. More fields means a narrower rule.

| Field | Type | Notes |
|---|---|---|
| `CDHash` | string | 40-character code directory hash. Pins one exact build. |
| `TeamID` | string | Code signing team identifier. For Apple binaries, which have an empty team ID, use the literal `*APPLE*` (asterisks included) instead of an empty string. |
| `SigningID` | string | Code signature signing identifier. |
| `PathPrefix` | string | File system path prefix. |
| `SigningState` | string | One of `All` (default), `TestFlight`, `DeveloperID`, `Enterprise`, `AppStore`, `Apple`. |

Apple's schema spells out which combinations are valid, and they differ between the two lists:

| List | Must include one of | May also include |
|---|---|---|
| `AllowedBinaries` | `CDHash` or `TeamID` | `SigningID`, `PathPrefix`, or `SigningState` |
| `DeniedBinaries` | `CDHash`, `TeamID`, or `SigningID` | `PathPrefix` or `SigningState` |

That gives you a tunable trust radius:

- `TeamID` only: trust everything a developer signs. Broad, low maintenance, survives updates.
- `TeamID` + `SigningID`: trust one product from that developer.
- `CDHash`: pin one exact build. Nothing else passes, including the next version of the same app.
- Add `PathPrefix`: additionally require that the binary live in an expected location.

> **Note:** Matching is per binary, not per app. Many apps are several signed executables: helpers, updaters, login items, and XPC services. Chrome, for example, launches `Google Chrome Helper` processes with their own signing IDs. An entry that pins `SigningID` to the main app's identifier may let the app open and then break it. For multi-binary apps, start with `TeamID` alone, or list every signing ID the app uses.

> **Warning:** Code without a team ID can only be allowed by `CDHash`. Every allow entry needs a `CDHash` or a `TeamID`. Ad-hoc signed code has no team ID. That includes Homebrew bottles and anything a developer builds locally with `go build`, `cargo build`, or Xcode's "Sign to Run Locally". The only way to allow it is by hash, and the hash changes on every `brew upgrade` or rebuild. In practice, an allow list and an unmanaged developer toolchain don't mix. Plan on deny lists (or no list) for engineering until you've solved that.

## Build the inventory before you write the policy

### Inspect a single Mac

On a single Mac, `codesign` gives you everything the schema needs:

```sh
codesign -dvvv /Applications/Slack.app
```

Use `TeamIdentifier` for `TeamID`, `Identifier` for `SigningID`, and `CDHash` for `CDHash`. Apple's documentation points to the same command.

### Query apps across your fleet

Across a fleet, query it instead. Fleet collects code signing data through osquery's `signature` table.

```sql
SELECT
  apps.name,
  apps.bundle_identifier,
  signature.team_identifier,
  signature.identifier AS signing_id,
  signature.cdhash,
  signature.arch,
  signature.authority
FROM apps
JOIN signature ON signature.path = apps.path;
```

Universal binaries return one row per architecture. Every macOS 27 Mac is Apple silicon, so if you pin by `CDHash`, use the `arm64` value (unless the binary runs under Rosetta).

Export the results and count hosts per team ID and signing ID. That gives you a draft allow list: every signing identity present on your Macs, ranked by how common it is. Anything with a team ID you don't recognize is worth a conversation before you turn enforcement on.

### Query standalone binaries

Apps are the easy half. The query above won't see command-line tools in `/usr/local/bin`, `/opt/homebrew`, `~/bin`, or project directories. Those are exactly what the new controls reach that older ones didn't. Sweep the usual locations with the `file` table:

```sql
SELECT
  file.path,
  signature.signed,
  signature.team_identifier,
  signature.identifier AS signing_id,
  signature.cdhash,
  signature.arch
FROM file
JOIN signature ON signature.path = file.path
WHERE file.path LIKE '/usr/local/bin/%'
   OR file.path LIKE '/opt/homebrew/bin/%'
   OR file.path LIKE '/Users/%/bin/%';
```

### Record what runs

A directory sweep only finds binaries where you thought to look. For the full picture, record what runs. The `es_process_events` table in osquery uses Endpoint Security to log every exec with its team ID, signing ID, and CDHash:

```sql
SELECT
  team_id,
  signing_id,
  cdhash,
  path,
  COUNT(*) AS launches
FROM es_process_events
WHERE event_type = 'exec'
  AND platform_binary = 0
GROUP BY team_id, signing_id, cdhash, path;
```

This requires enabling osquery's Endpoint Security and events support through agent options (`disable_events: false` and `disable_endpointsecurity: false`). You also need to grant osquery Full Disk Access. Scheduled over a few weeks, it's the closest thing to an audit mode you can get. More on that below.

## Start with a deny list

A deny list changes the least and can't lock anyone out of their own machine.

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

The first entry blocks everything a developer signs. The second blocks a tool by signing ID regardless of who signed it. The third blocks one product from one developer.

A deny list is for software you know about and don't want: unsanctioned remote access tools, a retired agent, or a specific vulnerable build.

> **Note:** A deny list isn't malware protection. Anyone can re-sign a binary with a different identifier, or ad-hoc sign it with no team ID at all, and walk past a deny entry.

## Write an allow list

```json
{
  "Type": "com.apple.configuration.app.settings",
  "Identifier": "com.example.binaries.allow",
  "Payload": {
    "Allowed": {
      "AlwaysAllowManagedApps": true,
      "AllowedBinaries": [
        {
          "TeamID": "*APPLE*"
        },
        {
          "TeamID": "EQHXZ8M8AV"
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

In order: everything Apple signs, everything Google signs, everything Adobe signs, and one pinned build of a command-line tool that has to live in `/usr/local/bin`.

With `AlwaysAllowManagedApps` set to `true`, everything you deploy through Fleet as managed software is already covered. The array only needs the things you *didn't* deploy.

Apple's own examples use `*APPLE*` with a `SigningID` to allow a single Apple component, for example `com.apple.Safari.WebApp` for Safari web apps. Since allow entries require `TeamID` or `CDHash`, `SigningState: "Apple"` can't stand on its own as a replacement for `*APPLE*`. It can only narrow an entry that already has one of those.

> **Note:** Most of the OS is exempt. Apple says most binaries in the signed, sealed system volume are always permitted, and the schema says the device always runs system-critical processes. The word *most* matters: Apple-signed code that lives outside the sealed volume, like Xcode, the iWork apps, or Safari web apps, still needs an allow entry.

## Deliver the declaration with Fleet

The declaration type only exists on macOS 27. Older Macs reject it, and Fleet shows the error in OS settings:

```
Error.UnknownDeclarationType: Unknown Declaration Type map[UnknownDeclarationType:com.apple.configuration.app.settings]
```

So scope it. In Fleet, go to **Labels**, add a dynamic label named `macOS 27+`, and use:

```sql
SELECT 1 FROM os_version WHERE major >= 27;
```

Then add the declaration and target it to hosts that have the `macOS 27+` label:

1. Save your declaration as a `.json` file.
2. Go to **Controls > OS settings > Configuration profiles**.
3. Select **Add profile** and upload the `.json` file.
4. Under **Target**, select **Include any** and choose the `macOS 27+` label.
5. Select **Save**.

If you manage Fleet with GitOps, keep the declaration in your repo alongside your other configuration profiles. That becomes the exception process later.

> **Warning:** Keep each Mac to one allow list. If a Mac receives more than one of these declarations, Apple merges `DeniedBinaries` as a union, `AlwaysAllowManagedApps` as a logical OR, and `AllowedBinaries` as an intersection. A second allow-list declaration narrows what can run. It doesn't add to it.

> **Note:** The `Privacy` key on the same declaration type is user scope on macOS, while the binary keys are system scope, so they need separate declarations. Fleet's support for user-scoped declarations is still in progress.

## Roll it out without breaking everyone

> **Warning:** There's no audit-only mode in Apple's schema. `AllowedBinaries` enforces the moment it lands.

Sequence the rollout yourself. Here's a suggested order:

1. Build the inventory. Run the signing queries above across the whole fleet, and collect `es_process_events` long enough to catch the once-a-quarter tools.
2. Deploy a deny list to production. Block the specific things you already know you don't want. It's low risk, delivers immediate value, and exercises the delivery path.
3. Deploy the allow list to a pilot label. Ten volunteers, `AlwaysAllowManagedApps: true`, team IDs from the inventory. Before you deploy, compare the pilot group's `es_process_events` history to the draft list. Anything that ran and isn't covered would have been blocked. Watch for a week.
4. Widen by role. Engineering will have the longest tail by a wide margin, and the ad-hoc signing problem above may rule out an allow list there entirely. Expect to iterate far more than on a standard laptop build.
5. Document the exception path. When someone gets blocked, they need a route that ends in a pull request, not a Slack DM to whoever is awake.

> **Note:** The user sees an alert when a launch is blocked, but it doesn't say much. Plan the support conversation before you ship.

## Related changes in macOS 27

Some PPPC services are deprecated in macOS 27. In `com.apple.TCC.configuration-profile-policy`, the `Accessibility`, `Camera`, `Microphone`, `SpeechRecognition`, and `BluetoothAlways` services are deprecated in favor of `Privacy` in `com.apple.configuration.app.settings`. Accessibility isn't one you can sit on. In macOS 27, the device shows a notification when the setting is applied and lets the user change it in System Settings, so it's no longer something you can enforce. That's a topic for a separate guide.

## Further reading

- [WWDC26 app management updates](https://support.apple.com/guide/deployment/app-management-updates-depd567c9ffa/web) (Apple Platform Deployment)
- [AppSettings](https://developer.apple.com/documentation/devicemanagement/appsettings) (Apple Developer Documentation)
- [AppSettingsAllowedObject](https://developer.apple.com/documentation/devicemanagement/appsettingsallowedobject) (Apple Developer Documentation)
- [AppSettingsAllowed_BinaryIdentifierObject](https://developer.apple.com/documentation/devicemanagement/appsettingsallowed_binaryidentifierobject) (Apple Developer Documentation)
- [app.settings.yaml schema](https://github.com/apple/device-management/blob/release/declarative/declarations/configurations/app.settings.yaml) (apple/device-management on GitHub)
- [What's new in managing Apple devices](https://developer.apple.com/videos/play/wwdc2026/206/) (WWDC26 session 206)
- [Configuration profiles](https://fleetdm.com/guides/custom-os-settings) (Fleet)
- [Declarative device management: a primer](https://fleetdm.com/articles/declarative-device-management-a-primer) (Fleet)

<meta name="articleTitle" value="Binary authorization on macOS 27: allow and deny lists with native Apple features">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="publishedOn" value="2026-09-28">
<meta name="category" value="guides">
<meta name="description" value="Use macOS 27's binary allow and deny lists to control which apps and command-line tools can run on your Macs, and deliver them with Fleet.">
