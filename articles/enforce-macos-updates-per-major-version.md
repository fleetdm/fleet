# Enforce macOS updates per major version using custom DDM declarations

Fleet's built-in OS update enforcement lets you set a single minimum patch version per fleet. That works fine when your fleet is on one major OS. It falls short when you're supporting a transition, like running macOS 15 and macOS 26 devices on the same fleet and needing different patch floors for each.

This guide shows how to work around that limitation using custom Apple DDM declarations scoped to labels.

## Prerequisites

- Fleet v4.86 or earlier: enable the `mdm.allow_all_declarations` feature flag on your Fleet server before following these steps. Set the environment variable `FLEET_MDM_ALLOW_ALL_DECLARATIONS=1` and restart your Fleet server. See [Fleet server configuration](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-allow-all-declarations) for details.
- Fleet v4.87 or later: this flag is enabled by default. No action needed.
- macOS 14 or later on managed devices (required by Apple for `softwareupdate.enforcement.specific` declarations).
- Fleet Premium (required for label-scoped profiles).

> **Warning:** Don't use this approach alongside Fleet's built-in minimum OS version enforcement for macOS. Both methods deploy `softwareupdate.enforcement.specific` declarations. Using both at once will result in conflicting declarations on your devices. If you follow this guide, clear the minimum OS version setting in **Controls > OS updates** for macOS.

## How it works

You'll create one DDM declaration per supported major OS version, each targeting a specific minimum patch version. Then you'll scope each declaration to a label that identifies devices running that major OS.

## Step 1: Create labels for each major OS version

Create a dynamic label for each major macOS version you need to support. In Fleet, go to **Labels** and create a new dynamic label using an osquery query.

For macOS 15 devices, for 15.7.7:

- Label name: `macOS 15 update`
- Query:

```sql
SELECT 1 FROM os_version
WHERE major = 15
  AND (
    minor < 7
    OR (minor = 7 AND patch < 7)
  );
```

For macOS 26 devices, for 26.5.1:

- Label name: `macOS 26 update`
- Query:

```sql
SELECT 1 FROM os_version
WHERE major = 26
  AND (
    minor < 5
    OR (minor = 5 AND patch < 1)
  );
```

If the profile only includes major and minor version numbers, you can simplify the query. For 26.5:

```sql
SELECT 1 FROM os_version WHERE major = 26 AND minor < 5;
```

Repeat for any other major versions you need to cover.

> **Note:** If you deploy a profile that targets a version that a device is already at or above, then the profile will fail to apply to the device. The error will look similar to this:
> 
> ```
> Error.ConfigurationCannotBeApplied: Configuration cannot be applied map[Error:[kSUCoreErrorDDMInvalidDeclarationFailure] Invalid declaration: target OS version (15.7) is older than current version (15.7.7)]
> ```
> 
> This means that the device is already compliant and no update is needed, but Fleet will show the profile status as "Failed".

## Step 2: Create a DDM declaration for each major OS version

Create a separate JSON file for each major version. Each file defines the minimum patch version you want to enforce and the deadline by which devices must comply.

Update `TargetOSVersion` to the minimum patch you want to enforce and `TargetLocalDateTime` to a deadline that gives your users adequate notice.

**macos-15-update-enforcement.json**

```json
{
  "Type": "com.apple.configuration.softwareupdate.enforcement.specific",
  "Identifier": "fleet.softwareupdate.macos15",
  "Payload": {
    "TargetOSVersion": "15.7.7",
    "TargetLocalDateTime": "2026-07-31T23:59:59"
  }
}
```

**macos-26-update-enforcement.json**

```json
{
  "Type": "com.apple.configuration.softwareupdate.enforcement.specific",
  "Identifier": "fleet.softwareupdate.macos26",
  "Payload": {
    "TargetOSVersion": "26.5.1",
    "TargetLocalDateTime": "2026-07-31T23:59:59"
  }
}
```

`TargetOSVersion` is the minimum patch version you want to enforce. `TargetLocalDateTime` is the local deadline on the device, after which macOS will force the update. Both fields are required for enforcement to take effect.

## Step 3: Upload declarations and scope them to labels

### Using the Fleet UI

1. Go to **Controls** > **OS settings**.
2. Click **Add profile**.
3. Upload `macos-15-update-enforcement.json`.
4. Set **Display name** to something descriptive, like `macOS 15 update enforcement`.
5. Under **Target**, select **Include any** and choose the `macOS 15 update` label.
6. Click **Save**.
7. Repeat for `macos-26-update-enforcement.json`, targeting the `macOS 26 update` label.

### Using GitOps

Add the declarations to your fleet YAML under `controls.macos_settings.custom_settings`, with label scoping:

```yaml
controls:
  macos_settings:
    custom_settings:
      - path: ./declarations/macos-15-update-enforcement.json
        labels_include_any:
          - macOS 15 update
      - path: ./declarations/macos-26-update-enforcement.json
        labels_include_any:
          - macOS 26 update
```

Run `fleetctl gitops` to apply.

## Verify

After Fleet delivers the declarations, check that each device received the correct one:

1. Go to **Hosts** and filter by the `macOS 15 update` label.
2. Click a host and go to the **OS settings** tab.
3. Confirm `macOS 15 update enforcement` shows as **Verified**.

Repeat for the `macOS 26 update` label.

Devices that don't match any label won't receive a software update declaration via this method. If you have devices on older major versions you want to force off, create an additional declaration (or use Fleet's built-in enforcement) targeting those devices.

## Related resources

- [Fleet server configuration: `mdm.allow_all_declarations`](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-allow-all-declarations)
- [Apple documentation: software update enforcement declaration](https://developer.apple.com/documentation/devicemanagement/softwareupdateenforcementspecific)
- [Configuration profiles in Fleet](https://fleetdm.com/docs/using-fleet/mdm-macos-settings)

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="authorFullName" value="Kitzy">
<meta name="publishedOn" value="2026-06-16">
<meta name="articleTitle" value="Enforce macOS updates per major version using custom DDM declarations">
<meta name="description" value="Enforce different minimum patch versions for major macOS versions on the same fleet using custom DDM declarations scoped to labels.">
