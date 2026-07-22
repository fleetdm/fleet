# Managed Migration Assistant: Mac-to-Mac migration with Fleet

Replacing a Mac means figuring out how to get the user's files to the new one without relying on them to do it manually. Apple's Managed Migration Assistant, introduced in macOS 26.4, lets your MDM specify what transfers from a user's Home folder during ADE enrollment. Fleet supports the `await_device_configured` key required to deliver the configuration at the right point in Setup Assistant, and the declarative status channel provides visibility during and after the transfer.

## Why managed migration keeps Fleet's MDM working

Migration Assistant has always carried a hidden risk for managed Macs. When a user migrates the standard way and leaves the **Other Files & Folders** option checked, macOS copies the old Mac's MDM enrollment state onto the new one. The new Mac ends up carrying the old device's configuration profiles and a stale enrollment with no valid identity. fleetd still checks in and osquery keeps working, so the host looks fine, but MDM breaks: commands hang or land on the wrong device.

Managed Migration Assistant closes that gap. The declaration tells macOS to migrate only the contents of the user's Home folder into the account created during Setup Assistant, so the new Mac keeps its own ADE enrollment and Fleet's MDM identity stays intact. Deploy the profile described below and your users can bring their data over without breaking fleetd or MDM, and without relying on anyone to remember to uncheck the right box.

## Requirements

Check these before deploying:

- The source Mac (the old one) must run macOS 15 or later.
- The destination Mac (the new one) must run macOS 26.4 or later.
- The destination Mac must be registered in Apple School or Apple Business and enrolled via Automated Device Enrollment. This configuration requires supervision and does not support other enrollment methods.
- Both Macs need a data connection. Migration Assistant uses peer-to-peer Wi-Fi when available and checks throughout the transfer for a faster option. It also supports infrastructure Wi-Fi, Ethernet, and Thunderbolt.

The source Mac requires no MDM configuration. Nothing needs to be deployed to it in advance.

## What transfers and what doesn't

Managed Migration Assistant works within the user's Home folder. Here's what it can transfer:

- Visible folders in the Home folder
- Hidden folders and files (`.ssh`, `.bash_history`, and similar)
- Folder aliases and symlinks inside the Home folder (originals outside the Home folder won't transfer)
- Privacy and security settings

The following are not available for migration:

- Applications (`/Applications`)
- Files and folders in `/Users/Shared/`
- File aliases and symlinks in the Home folder
- Printers and services
- Other system settings

The difference between the two alias and symlink entries above is intentional: Apple transfers folder-level aliases and symlinks but not file-level ones. If a symlink points to a file rather than a folder, it won't move.

The `~/Library` folder always transfers. You cannot exclude it.

Plan to handle applications, security tooling, and system configuration through Fleet separately. The migration delivers the user's files. Fleet handles everything else.

## Configure Managed Migration Assistant in Fleet

The declaration type is `com.apple.configuration.migration-assistant.settings`. The declaration file is the same whether you use GitOps or the Fleet UI. Here's an example to start from:

```json
{
  "Type": "com.apple.configuration.migration-assistant.settings",
  "Identifier": "com.example.migration-assistant",
  "Payload": {
    "ShouldDoManagedMigration": true,
    "ShouldMigrateSecurityPrivacySettings": true,
    "RequiredPaths": [
      "Desktop/",
      "Documents/"
    ],
    "ExcludedPaths": [
      "Downloads/",
      ".Trash/"
    ]
  }
}
```

`ShouldDoManagedMigration` and `ShouldMigrateSecurityPrivacySettings` set to `true` are the baseline that makes migration safe for a managed Mac; the two must both be present. `RequiredPaths` and `ExcludedPaths` are optional and only shape which folders move. A profile with just the two boolean keys is a valid, complete configuration.

A few things to know about paths before you customize:

- Paths are relative to the user's Home folder. To require `~/Documents/Work/`, specify `Documents/Work/`.
- Folder paths require a trailing slash (`/`).
- You can combine `RequiredPaths` and `ExcludedPaths`. Requiring `Documents/` and excluding `Documents/Archive/` is valid.
- Order matters in `RequiredPaths`. If the destination Mac runs low on storage, priority follows the order you listed.
- Hidden paths work in both arrays. To exclude `.Trash`, specify `.Trash/`.

After the user account is created, Managed Migration Assistant presents the user with the transfer interface. Required paths appear pre-selected and cannot be deselected. Excluded paths don't appear at all.

One constraint from Apple: the **Restore** pane in Setup Assistant cannot be hidden when this feature is active. The `Restore` skip key has no effect here.

### Scope the profile to macOS 26.4 or later

The `com.apple.configuration.migration-assistant.settings` declaration type only exists on macOS 26.4 and later. If Fleet delivers it to an older Mac, the profile fails and the host's OS settings show:

```
Error.UnknownDeclarationType: Unknown Declaration Type map[UnknownDeclarationType:com.apple.configuration.migration-assistant.settings]
```

Scope the profile to a dynamic label so only eligible hosts receive it. In Fleet, go to **Labels**, add a dynamic label named something like `macOS 26.4+`, and use this query:

```sql
SELECT 1 FROM os_version WHERE major > 26 OR (major = 26 AND minor >= 4);
```

Then target that label when you add the profile, as shown below.

### GitOps

1. Save your declaration as a `.json` file in your repository.

2. Reference it under `controls.macos_settings.custom_settings` in your team YAML, scoped to the label:

```yaml
controls:
  macos_settings:
    custom_settings:
      - path: ./platforms/macos/declaration-profiles/migration-assistant.json
        labels_include_any:
          - macOS 26.4+
```

3. Commit and push. Your CI/CD pipeline will run `fleetctl gitops` and apply the declaration.

### Fleet UI

1. Save your declaration as a `.json` file.

2. In the Fleet UI, navigate to **Controls > OS settings > Configuration profiles**.

3. Select the fleet you want to add the profile to.

4. Select **Add profile** and upload your `.json` file.

5. Under **Target**, select **Include any** and choose the `macOS 26.4+` label.

6. Select **Save**.

Fleet will deliver the declaration to the supervised, ADE-enrolled macOS hosts in that fleet that match the label.

For a full reference of the declaration schema, see the [Apple Platform Deployment guide](https://support.apple.com/guide/deployment/managed-migration-assistant-for-macos-dep4f861792f/web) and the [apple/device-management](https://github.com/apple/device-management/blob/release/declarative/declarations/configurations/migration-assistant.settings.yaml) GitHub repo.

## Handle standard user authentication

Migration Assistant on the source Mac requires the user to authenticate with local administrator credentials before the transfer starts. If your users are standard users, they can't launch it without help.

If elevating users to admin isn't possible in your environment, you can modify the `authorizationdb` to allow standard users to authenticate Migration Assistant with their own credentials instead of an admin password.

Run this on the source Mac before the migration:

```bash
sudo security authorizationdb write com.apple.system-migration.launch-password authenticate-session-owner
```

This swaps the admin authentication prompt for a user-level authentication dialog.

After the migration completes, reset it to the default:

```bash
sudo security authorizationdb write com.apple.system-migration.launch-password authenticate-admin-nonshared-password
```

## End-to-end flow

Once everything is configured, here's what the process looks like:

1. The user opens Migration Assistant on the source Mac and authenticates.
2. The user powers on the new Mac and begins Setup Assistant.
3. On the **Transfer Your Data to This Mac** pane, the user selects the source Mac.
4. The new Mac enrolls in Fleet via ADE. Fleet delivers the migration declaration.
5. After the user account is created, Managed Migration Assistant presents the transfer interface with your configured paths.
6. The transfer begins. Fleet reports status through the declarative status channel.
7. Migration completes. Fleet delivers a post-transfer report.

Both Macs need to stay within range of each other until the transfer finishes.

## Further reading

- [Managed Migration Assistant for macOS — Apple Platform Deployment](https://support.apple.com/guide/deployment/managed-migration-assistant-for-macos-dep4f861792f/web)
- [Managed Migration Assistant declarative configuration — Apple Platform Deployment](https://support.apple.com/guide/deployment/managed-migration-assistant-declarative-depd18014adc/1/web/1.0)
- [Managed Migration Assistant — Magic That Works](https://magicthatworks.net/blog/managed-migration-assistant/) (Adam Selby's testing and notes)
- [apple/device-management — migration-assistant.settings.yaml](https://github.com/apple/device-management/blob/release/declarative/declarations/configurations/migration-assistant.settings.yaml)

<meta name="articleTitle" value="Managed Migration Assistant: Mac-to-Mac migration with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-06-26">
<meta name="category" value="guides">
<meta name="description" value="macOS 26.4 adds Managed Migration Assistant. Learn how to configure it with Fleet to control what transfers during Mac-to-Mac migrations.">
