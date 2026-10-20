# Show Nudge only after macOS prepares an update

When Nudge asks someone to update before macOS has downloaded the update, clicking **Update** can mean a long wait. This guide sets up Fleet so Nudge's settings install only after macOS reports the update as prepared. When end users click **Update**, all that's left is the restart. The approach comes from Wes Whetstone's MacDevOpsYUL 2026 talk, [But you called me?](https://www.youtube.com/watch?v=SggxxPCDCSI). For the background and the reasoning, read the companion article, [Don't put users on hold: show Nudge when the update is ready](https://fleetdm.com/articles/dont-put-users-on-hold-show-nudge-when-the-update-is-ready).

Every file in this guide is based on Fleet's own configuration in the [`it-and-security` folder](https://github.com/fleetdm/fleet/tree/main/it-and-security) of the Fleet repository.

## Prerequisites

- Fleet 4.91.0 or later, with `fleetctl` 4.91.0 or later if you use GitOps. Earlier versions don't support custom activations.
- Apple MDM turned on in Fleet.
- Macs on macOS 14 or later, enrolled with Automated Device Enrollment (ADE). The `softwareupdate.install-state` status item is only available on supervised Macs.
- Nudge installed on your Macs, with its launch agent loaded. See [Enforce OS updates on macOS with Nudge](https://fleetdm.com/guides/enforce-os-updates-on-macos-with-nudge).
- A Nudge configuration profile (`.mobileconfig`) you're happy with.
- A place to host that profile at an `https://` URL.

## Step 1: Turn off Software Update notifications

Nudge should be the only update prompt your users see. Add a software update settings declaration that turns off Software Update's notifications and keeps automatic downloads on. Fleet's version is [`Software Update settings.json`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/declaration-profiles/Software%20Update%20settings.json):

```json
{
    "Type": "com.apple.configuration.softwareupdate.settings",
    "Identifier": "com.example.config.softwareupdate.settings",
    "Payload": {
        "AutomaticActions": {
            "Download": "AlwaysOn"
        },
        "Notifications": false
    }
}
```

With automatic downloads on, macOS downloads and prepares updates in the background. To make sure updates are downloaded by a specific date, also set a minimum version and deadline in Fleet's [OS update settings](https://fleetdm.com/guides/enforce-os-updates). Fleet then sends macOS a software update enforcement declaration, which downloads and prepares the update.

> **Warning:** When a Mac receives an enforcement declaration, it starts downloading the update right away. If you send it to a large fleet at once, every Mac downloads at the same time. Roll it out in waves, using labels or fleets, if your network can't absorb that.

## Step 2: Split Nudge's profile

If your Nudge profile includes a `com.apple.servicemanagement` payload that keeps Nudge's launch agent a managed login item, move that payload into its own profile. Otherwise it's installed and removed with every update, and users see a "Background Items Added" notification each time.

1. Create a profile that contains only the `com.apple.servicemanagement` payload for `com.github.macadmins.Nudge`. Fleet's version is [`nudge-background-task.mobileconfig`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/configuration-profiles/nudge-background-task.mobileconfig).
2. Remove that payload from your Nudge profile, so the Nudge profile contains only the `com.github.macadmins.Nudge` payload. Fleet's version is [`nudge-preferences.mobileconfig`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/configuration-profiles/nudge-preferences.mobileconfig).
3. Give the Nudge profile a new `PayloadIdentifier` and new `PayloadUUID` values.

> **Note:** If Fleet already delivers your Nudge profile, applying this change removes the old profile from Macs. MDM commands and declarations are delivered separately, so a removal could delete the new profile if it reused the old identifier. A new identifier avoids that.

## Step 3: Host the Nudge profile

Upload the Nudge profile from step 2 to an `https://` URL your Macs can reach. Fleet doesn't host mobileconfig files, so Fleet's configuration uses the file's GitHub URL, pinned to a commit:

```
https://raw.githubusercontent.com/<org>/<repo>/<commit SHA>/path/to/nudge-preferences.mobileconfig
```

Macs download the profile again only when the declaration in step 4 changes. A URL that always points at the latest version, like a branch name, means settings changes never reach Macs that already have the profile. Pin the URL to a specific commit or version instead.

> **Note:** Apple describes the legacy profile URL as expected to be on the MDM server, and the Mac presents its device identity certificate when it downloads the file. Test the URL on a Mac before rolling out. If you host the file on your own server, see [How to secure externally hosted DDM assets](https://fleetdm.com/guides/securing-externally-hosted-ddm-assets) to verify those requests.

## Step 4: Create the legacy profile declaration

Create a declaration that tells the Mac to download and install the hosted profile. Fleet's version is [`Nudge settings.json`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/declaration-profiles/Nudge%20settings.json):

```json
{
    "Type": "com.apple.configuration.legacy",
    "Identifier": "com.example.config.nudge.legacy",
    "Payload": {
        "ProfileURL": "https://raw.githubusercontent.com/<org>/<repo>/<commit SHA>/path/to/nudge-preferences.mobileconfig"
    }
}
```

## Step 5: Create the activation

Create an activation that installs the declaration from step 4 only when the update is prepared. Fleet's version is [`Nudge activation.json`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/declaration-profiles/Nudge%20activation.json):

```json
{
    "Type": "com.apple.activation.simple",
    "Identifier": "com.example.activation.nudge",
    "Payload": {
        "StandardConfigurations": [
            "com.example.config.nudge.legacy"
        ],
        "Predicate": "@status(softwareupdate.install-state) == 'prepared'"
    }
}
```

`StandardConfigurations` must list the `Identifier` from step 4 exactly. The value `prepared` is lowercase.

## Step 6: Add the files to Fleet

Add the profiles and the declaration to your fleet's YAML file under `controls.apple_settings.configuration_profiles`, and attach the activation with the `activation` key. Fleet's version is in [`workstations.yml`](https://github.com/fleetdm/fleet/blob/main/it-and-security/fleets/workstations.yml):

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos/declaration-profiles/Software Update settings.json
      - path: ../lib/macos/configuration-profiles/nudge-background-task.mobileconfig
      - path: ../lib/macos/declaration-profiles/Nudge settings.json
        activation: ../lib/macos/declaration-profiles/Nudge activation.json
```

Don't add the Nudge profile from step 2 to this list. Macs download it from the URL in step 4.

Run `fleetctl gitops` or let your CI pipeline apply the change.

### Optional: Keep Nudge on Macs the activation can't reach

Two groups of Macs never report `prepared`:

- Macs that aren't enrolled with ADE, because they aren't supervised.
- Macs that need a major upgrade, if nothing makes macOS download that upgrade on its own.

To keep Nudge running on those Macs, give them a copy of the Nudge profile without the activation:

1. Create labels for those Macs. Fleet's versions are [macOS 15 and older hosts](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/all/labels/macos-15-and-older-hosts.yml) and [Macs not enrolled through ADE](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/all/labels/macs-not-enrolled-through-ade.yml).
2. Copy the Nudge profile from step 2 and give the copy its own `PayloadIdentifier`, `PayloadUUID` values, and `PayloadDisplayName`. Fleet's version is [`nudge-preferences-ungated.mobileconfig`](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/configuration-profiles/nudge-preferences-ungated.mobileconfig).
3. Exclude the labels from the declaration and target them with the copy:

```yaml
      - path: ../lib/macos/declaration-profiles/Nudge settings.json
        activation: ../lib/macos/declaration-profiles/Nudge activation.json
        labels_exclude_any:
          - "macOS 15 and older hosts"
          - "Macs not enrolled through ADE"
      - path: ../lib/macos/configuration-profiles/nudge-preferences-ungated.mobileconfig
        labels_include_any:
          - "macOS 15 and older hosts"
          - "Macs not enrolled through ADE"
```

> **Note:** Keep both copies' Nudge settings identical. When you change one, change the other in the same commit.

## Update Nudge's settings

Because the URL is pinned to a commit, changing Nudge's settings takes two commits:

1. Edit the Nudge profile, and the ungated copy if you have one, then commit.
2. Update the commit SHA in the declaration's `ProfileURL` to the commit from the first step, then commit again.

A commit can't contain its own SHA, so the two steps can't be combined. Don't rebase or amend the first commit after pinning it, because that changes its SHA.

## Verify

1. In Fleet, open a Mac's host details page and check the declaration's status under **OS settings**. Until an update is prepared, the declaration shows as **Verified**, with a note that the predicate evaluated to false.
2. On a Mac with a prepared update, open **System Settings** > **General** > **Device Management** and confirm the Nudge profile is listed.
3. Confirm Nudge opens and that clicking **Update** goes straight to the restart.

## Troubleshooting

**Nudge never appears, and the declaration shows as Verified.** The predicate is false. Check that the Mac is enrolled with ADE, runs macOS 14 or later, and has an update downloaded and prepared. If updates aren't being prepared, set a minimum version and deadline in Fleet's OS update settings.

**The declaration shows as Failed.** The Mac couldn't download or install the profile. Check that the `ProfileURL` loads from the Mac's network and returns a valid configuration profile.

**Settings changes don't reach Macs.** The `ProfileURL` still points at the old commit. Update the SHA in the declaration, as described in [Update Nudge's settings](#update-nudges-settings).

**GitOps fails with `unknown key` for `activation`.** Your `fleetctl` is older than 4.91.0. Update `fleetctl` to match your Fleet server.

## Further reading

- [Wes Whetstone, "But you called me?" at MacDevOpsYUL 2026](https://www.youtube.com/watch?v=SggxxPCDCSI)
- [Fleet's pull request that rolled this out on Fleet's own Macs](https://github.com/fleetdm/fleet/pull/51546)
- [Apple: legacy profile declaration](https://github.com/apple/device-management/blob/release/declarative/declarations/configurations/legacy.yaml)
- [Apple: `softwareupdate.install-state` status item](https://github.com/apple/device-management/blob/release/declarative/status/softwareupdate.install-state.yaml)
- [Enforce macOS updates per major version using custom DDM declarations](https://fleetdm.com/guides/enforce-macos-updates-per-major-version)

<meta name="articleTitle" value="Show Nudge only after macOS prepares an update">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-20">
<meta name="description" value="Use a DDM activation in Fleet so Nudge's settings install only after macOS has downloaded and prepared an update.">
