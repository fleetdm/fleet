# Migrating Intune policies to Fleet with the CSP converter

Migrating Windows configuration policies from Microsoft Intune to Fleet doesn't have to be a manual, policy-by-policy rebuild. The Intune-to-Fleet CSP converter is a community PowerShell tool that automates most of the translation work — converting your Intune JSON exports into [SyncML](https://en.wikipedia.org/wiki/SyncML) XML files ready to upload to Fleet.

> **Important:** This is a community tool, not an official Fleet product. It covers approximately 70–75% of standard Intune policy scenarios out of the box. Edge cases, custom or complex policies, and certain ADMX-backed configurations may require manual review or adjustment. Your mileage will vary — treat the output as a strong starting point, not a finished migration.

If you're new to Windows CSPs in Fleet, start with [Creating Windows configuration profiles (CSPs)](https://fleetdm.com/guides/creating-windows-csps) first. That guide explains the SyncML format, registry lookups, and how Fleet applies profiles to devices.

## How it works

The converter takes an Intune JSON policy export and produces SyncML XML files for each policy setting, ready to upload to Fleet as [custom OS settings](https://fleetdm.com/guides/custom-os-settings).

It does this in four steps:

1. **Policy extraction** — Recursively parses the Intune JSON, including nested choice settings and their child configurations.
2. **Registry lookup** — Queries the Windows CSP NodeCache registry (`HKLM:\SOFTWARE\Microsoft\Provisioning\NodeCache\CSP\Device\MS DM Server\Nodes`) to find the exact `NodeURI` path with proper TitleCase casing — which Windows CSPs require.
3. **Format detection** — Determines whether each setting should use `bool`, `int`, or `chr` SyncML format, based on the policy type and Microsoft's CSP documentation.
4. **Value resolution** — For policies where Intune stores `ExpectedValue = -1` (unset), a resolver map of PowerShell expressions queries the current system state to determine the correct value.

Each setting becomes a SyncML `<Replace>` block:

```xml
<Replace>
    <Item>
        <Meta>
            <Format xmlns="syncml:metinf">bool</Format>
        </Meta>
        <Target>
            <LocURI>./Vendor/MSFT/Firewall/MdmStore/PrivateProfile/EnableFirewall</LocURI>
        </Target>
        <Data>true</Data>
    </Item>
</Replace>
```

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

### Common usage patterns

Basic conversion (individual XML files per policy):
```powershell
.\Convert-IntuneToFleetCSP.ps1 -JsonPath "MyFirewallPolicy.json"
```

Create a single merged XML file:
```powershell
.\Convert-IntuneToFleetCSP.ps1 -JsonPath "MyPolicy.json" -MergeXml -OutputPath "C:\Fleet\CSPs"
```

Dry run — analyze without creating files:
```powershell
.\Convert-IntuneToFleetCSP.ps1 -JsonPath "MyPolicy.json" -DryRun
```

Debug mode — verbose output for troubleshooting:
```powershell
.\Convert-IntuneToFleetCSP.ps1 -JsonPath "MyPolicy.json" -DebugMode
```

### Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `JsonPath` | Path to Intune policy JSON export | *Required* |
| `ResolverMapPath` | Path to `resolver-map.json` | `C:\CSPConverter\resolver-map.json` |
| `OutputPath` | Directory for output XML files | `C:\CSPConverter\Output` |
| `LogPath` | Path for conversion log CSV | `C:\CSPConverter\ConversionLog.csv` |
| `DebugMode` | Verbose debug output | `$false` |
| `DryRun` | Analyze only, no files created | `$false` |
| `MergeXml` | Merge all policies into one XML file | `$false` |

### Step 3: Review the results

### Console summary

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

### Conversion log

A CSV log is saved to `C:\CSPConverter\ConversionLog.csv` (or your specified `-LogPath`). Open it in Excel or any CSV viewer to review every policy:

| Column | Description |
|--------|-------------|
| `Setting` | Original Intune `settingDefinitionId` |
| `NodeUri` | The resolved CSP path used in the XML |
| `Status` | Exported / Resolved / Skipped / Not Found / Error |
| `Value` | The data value written into `<Data>` |
| `Notes` | Format type and any resolution details |

Review `Not Found` and `Skipped` rows carefully — these are the policies that didn't convert and will need manual attention.

### Output files

Individual XML files are saved to `C:\CSPConverter\Output\` by default. Each file is named after the sanitized `NodeUri` path. If you used `-MergeXml`, you'll find a single `MergedPolicies.xml` instead.

### Step 4: Upload to Fleet

1. Review each output XML file. Verify the `<LocURI>` and `<Data>` values look correct for your environment.
2. In Fleet, navigate to **Controls** > **OS settings** > **Custom settings**.
3. Upload each XML file (or the merged file) and assign it to the appropriate team or hosts.

Fleet will deploy the profile to matching Windows devices on their next MDM check-in.

> **Tip:** Use a small test group or canary team in Fleet before rolling out converted profiles broadly. Even a "Resolved" policy may need tweaking if the resolver returned a value that doesn't match your intended configuration.

## Resolver map

The `resolver-map.json` file handles the `ExpectedValue = -1` edge case — policies where Intune didn't explicitly store the enforced value in the registry. The resolver map contains PowerShell expressions that query the current system state at conversion time.

Built-in resolvers cover common scenarios including:
- Windows Firewall profiles (`EnableFirewall`, `AllowLocalPolicyMerge`, etc.)
- Defender settings (`RealTimeProtection`, `CloudProtection`, `TamperProtection`)
- BitLocker (`RequireDeviceEncryption`, `RequireTPM`)
- SmartScreen, DeviceLock, Privacy, Update, and System policies

**To add a custom resolver**, add an entry to `resolver-map.json` where the key is the last segment of the CSP path and the value is a PowerShell expression returning `1` (enabled) or `0` (disabled):

```json
{
  "YourPolicyName": "try { if ((Get-ItemPropertyValue -Path 'HKLM:\\...' -Name 'PolicyValue') -eq 1) { 1 } else { 0 } } catch { 0 }"
}
```

## Extending the tool

### Adding boolean format policies

Most policies use integer format (`0` / `1`), but some Windows CSPs require `true` / `false` boolean values. If you find a policy is being written as `int` when it should be `bool`, add a wildcard pattern to the `$booleanFormatPolicies` array in the `Get-SyncMLFormatAndData` function:

```powershell
$booleanFormatPolicies = @(
    # ... existing patterns ...
    "*your*custom*policy*pattern*"
)
```

## Troubleshooting

### "No match found for NodeUri"

The policy wasn't found in the CSP NodeCache registry. This means either:
- The policy isn't supported on your Windows version.
- The policy was never applied to the device via Intune (the NodeCache is only populated for settings that have been pushed and applied).

Try running the script on a Windows host where that specific policy has been actively enforced by Intune.

### "Resolver execution failed"

A resolver map entry failed to execute. Check that:
- The required PowerShell modules are installed (e.g., `Get-MpPreference` requires Windows Defender).
- The PowerShell expression syntax in `resolver-map.json` is valid.
- The script is running with sufficient privileges.

### Policies convert with unexpected values

Use debug mode to trace the processing of a specific policy:

```powershell
.\Convert-IntuneToFleetCSP.ps1 -JsonPath "MyPolicy.json" -DebugMode -DryRun
```

Debug output shows the registry lookup result, format detection logic, and the value determination for each setting. No files are created when combined with `-DryRun`.

### Profile fails to apply in Fleet

If Fleet reports a profile error after upload, check the Windows MDM event log on an affected device:

```powershell
Get-WinEvent -FilterHashtable @{LogName='Microsoft-Windows-DeviceManagement-Enterprise-Diagnostics-Provider/Admin'; Level=2} -MaxEvents 15 | Format-Table -Wrap
```

The most common cause is a `<Format>` mismatch — the CSP expects `bool` but received `int`, or vice versa. Cross-reference the CSP path in [Microsoft's documentation](https://learn.microsoft.com/en-us/windows/client-management/mdm/) to confirm the expected format.

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
