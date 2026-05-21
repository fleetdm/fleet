# Migrating Intune policies to Fleet with the CSP converter

Migrating Windows configuration policies from Microsoft Intune to Fleet doesn't have to be a manual, policy-by-policy rebuild. The Intune-to-Fleet CSP converter is a community PowerShell tool that automates most of the translation work — converting your Intune JSON exports into [SyncML](https://en.wikipedia.org/wiki/SyncML) XML files ready to upload to Fleet.

> **Important:** This is a community tool, not an official Fleet product. It covers approximately 70–75% of standard Intune policy scenarios out of the box. Edge cases, custom or complex policies, and certain ADMX-backed configurations may require manual review or adjustment. Your mileage will vary — treat the output as a strong starting point, not a finished migration.

If you're new to Windows CSPs in Fleet, start with [Creating Windows configuration profiles (CSPs)](https://fleetdm.com/guides/creating-windows-csps) first. That guide explains the SyncML format, registry lookups, and how Fleet applies profiles to devices.

## Prerequisites

Before running the converter:

- **PowerShell 5.1 or later** — Built into Windows 10 or 11 by default.
- **Windows system enrolled in Intune** — The converter queries the CSP NodeCache registry, which only exists on devices that have received policies from Intune. You must run it on a managed Windows host.
- **Administrative rights** — Recommended for full registry access. Some lookups may fail or fall back to defaults without elevation.
- **`resolver-map.json`** — The companion file that ships with the tool. Place it at `C:\CSPConverter\resolver-map.json` (the default location) or pass the path explicitly with `-ResolverMapPath`.

## Download the tool

Get the latest version from GitHub:

- [`Convert-IntuneToFleetCSP.ps1`](https://github.com/tux234/intune-to-fleet/blob/main/Convert-IntuneToFleetCSP.ps1)
- [`resolver-map.json`](https://github.com/tux234/intune-to-fleet/blob/main/resolver-map.json)

Place both files in the same directory on your Windows host.

### Step 1: Export your Intune policy

1. Open the [Microsoft Intune Admin Center](https://intune.microsoft.com/#view/Microsoft_Intune_DeviceSettings/DevicesMenu/~/configuration).
2. Navigate to **Devices** > **Configuration**.
3. Select the policy you want to migrate.
4. Click the three-dot menu (**...**) and select **Export JSON**.
5. Save the file to your Windows host.

### Step 2: Run the converter

Open PowerShell as Administrator on your Intune-enrolled Windows host and run:
```powershell
.\Convert-IntuneToFleetCSP.ps1 -JsonPath "C:\Path\To\YourPolicy.json"
```

### Step 3: Review the results

When the script finishes, it prints a summary:

```
===== CONVERSION SUMMARY =====
Total Policies: 24
Exported:       18
Resolved:        3
Skipped:         1
Not Found:       2
===============================
```

| Status | Meaning |
|--------|---------|
| Exported | Policy converted successfully |
| Resolved | `ExpectedValue = -1` in registry; actual value determined via resolver |
| Skipped | No value could be determined (no resolver, no usable suffix) |
| Not Found | No matching entry in the CSP NodeCache registry |


Individual XML files are saved to `C:\CSPConverter\Output\` by default. Each file is named after the sanitized `NodeUri` path. If you used `-MergeXml`, you'll find a single `MergedPolicies.xml` instead.

> **Important:** For troubleshooting tips and more information on how this works, check out the [`intune-to-fleet` README](https://github.com/tux234/intune-to-fleet).

### Step 4: Upload to Fleet

1. Review each output XML file. Verify the `<LocURI>` and `<Data>` values look correct for your environment.
2. In Fleet, navigate to **Controls** > **OS settings** > **Custom settings**.
3. Upload each XML file (or the merged file) and assign it to the appropriate team or hosts.

Fleet will deploy the profile to matching Windows devices on their next MDM check-in.

> Use a small test group or canary team in Fleet before rolling out converted profiles broadly. Even a "Resolved" policy may need tweaking if the resolver returned a value that doesn't match your intended configuration.


## What this tool doesn't cover

Not all Intune configurations translate directly. These scenarios will appear as `Not Found` or `Skipped` in the log and need manual handling:

- **ADMX-backed policies** — Policies that require ADMX ingestion use `chr` format with CDATA payloads. The [Creating Windows CSPs guide](https://fleetdm.com/guides/creating-windows-csps) covers how to build these manually.
- **Settings Catalog policies with complex multi-value structures** — Some policies have nested group settings that aren't supported by the converter's current parsing logic.
- **Custom OMA-URI configurations** — These are already in SyncML format in Intune but may use non-standard CSP paths the NodeCache doesn't index.
- **App configuration policies** — The converter handles device configuration policies only.

## Conclusion

The CSP converter can significantly reduce the time it takes to migrate your Intune policy baseline to Fleet. In practice, expect to convert the majority of your standard settings automatically and spend the remaining effort on edge cases, ADMX-backed configurations, and validation.

Use the conversion log and debug mode to understand what converted cleanly and what needs manual follow-up. When in doubt, test on a small group of devices in Fleet before rolling changes to your full fleet.

<meta name="articleTitle" value="Migrating Intune policies to Fleet with the CSP converter">
<meta name="authorFullName" value="Mitch Francese">
<meta name="authorGitHubUsername" value="Tux234">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-03-06">
<meta name="description" value="Use the open-source Intune-to-Fleet CSP converter to automate migration of your Windows Intune policies to Fleet configuration profiles.">
