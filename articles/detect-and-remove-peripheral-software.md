# Detect and remove unwanted software installed by peripherals in Fleet

When you plug in a monitor, docking station, or printer, Windows can quietly install companion software — from monitor utilities that push trialware ads to docking station managers nobody asked for. The recent LG Monitor App story (July 2026) made headlines, but the broader problem is older: peripherals bundling junkware that lands on corporate workstations.

This guide walks through detecting and automatically removing unwanted software using Fleet policies and scripts. It covers traditional Windows installers (MSI/EXE) and Store-installed MSIX apps like the LG Monitor App.

## Prerequisites

Check these before you start:

- **Fleet Premium** for policy automation scripts. Script automations (triggering remediation when a policy fails) require Fleet Premium. Fleet Free users can create detection policies and run scripts manually from the UI.
- **Windows hosts enrolled in Fleet.** Detection uses the `programs` table for both MSI/EXE software and Store (MSIX) apps.
- **osquery 5.22.1 or later** for full Store app visibility. osquery 5.17.0 added MSIX packages to the `programs` table, and 5.22.1 fixed a gap where provisioned or never-launched Store apps were invisible. Check your agents with a live query: `SELECT version FROM osquery_info;`.
- **Scripts enabled.** If you use Fleet's MDM features, scripts are enabled by default. If you deploy fleetd without MDM, pass the `--enable-scripts` flag during installation.

## Create a policy to detect unwanted software (MSI/EXE)

In Fleet, a policy passes when its query returns at least one row, and fails when it returns zero rows. To detect unwanted software, invert the logic: return a row when the software is NOT present.

1. Navigate to **Policies** and click **Add policy**.
2. In the **Name** field, enter "McAfee trial software detected."
3. In the **Query** field, paste the following SQL:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM programs WHERE name LIKE '%McAfee%'
);
```

4. In the **Resolution** field, add instructions for your help desk: "McAfee trial software was detected and has been automatically removed. Contact IT if you were expecting to use McAfee products on this machine."

This query returns a row (pass) when no McAfee software exists. When McAfee is found, the subquery returns results, `NOT EXISTS` evaluates to false, and the outer query returns zero rows — the policy fails and triggers any attached automation.

> **Note:** The `programs` table reads the Windows Uninstall registry keys (both MSI and EXE installers that register in Add/Remove Programs). On osquery 5.17.0 and later it also includes Store (MSIX) apps — see the Store apps section below for how to target those precisely.

## Create a script to remove unwanted software

1. Navigate to **Controls > Scripts** and click **Add script**.
2. Name the script "Remove McAfee trial software" and set **Platform** to Windows.
3. In the **Script** field, paste the following PowerShell:

```powershell
# Find McAfee entries in the Windows Uninstall registry and remove them
$uninstallPaths = @(
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
    "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall"
)

$found = $false
$failed = $false

# Exit codes that indicate a successful uninstall:
# 0 = success, 3010 = success but reboot required (common with /norestart),
# 1641 = success, reboot initiated by the installer.
$successCodes = @(0, 3010, 1641)

# Parse an uninstall string into executable + arguments and run it.
# Handles quoted paths with spaces (e.g. "C:\Program Files\...\uninstall.exe" /S)
# without laundering the command through cmd.exe.
function Invoke-Uninstaller {
    param(
        [string]$CommandLine,
        [string]$ExtraArgs = ""
    )

    if ($CommandLine -match '^"([^"]+)"\s*(.*)$') {
        $exe = $matches[1]
        $argString = $matches[2]
    } else {
        $parts = $CommandLine -split '\s+', 2
        $exe = $parts[0]
        $argString = if ($parts.Count -gt 1) { $parts[1] } else { "" }
    }

    if ($ExtraArgs) {
        $argString = ("$argString $ExtraArgs").Trim()
    }

    if ([string]::IsNullOrWhiteSpace($argString)) {
        $proc = Start-Process -FilePath $exe -NoNewWindow -Wait -PassThru
    } else {
        $proc = Start-Process -FilePath $exe -ArgumentList $argString -NoNewWindow -Wait -PassThru
    }
    return $proc.ExitCode
}

foreach ($basePath in $uninstallPaths) {
    if (-not (Test-Path $basePath)) { continue }

    Get-ChildItem $basePath | ForEach-Object {
        $entry = Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue
        if (-not $entry) { return }
        if ($entry.DisplayName -notlike '*McAfee*') { return }

        $found = $true
        $name = $entry.DisplayName
        Write-Output "Found: $name"

        # MSI uninstall: rebuild the command from the product GUID.
        # Matches both /I{GUID} (modify) and /X{GUID} (uninstall) registrations,
        # including quoted msiexec paths like "C:\Windows\system32\msiexec.exe" /I{GUID}.
        if ($entry.UninstallString -match 'MsiExec\.exe"?\s*/(I|X)\s*\{(.+?)\}') {
            $guid = $matches[2]
            Write-Output "Uninstalling MSI: $name"
            $proc = Start-Process "MsiExec.exe" -ArgumentList "/X{$guid} /qn /norestart" -NoNewWindow -Wait -PassThru
            if ($proc.ExitCode -notin $successCodes) {
                Write-Output "Failed: exit code $($proc.ExitCode)"
                $failed = $true
            } elseif ($proc.ExitCode -eq 3010) {
                Write-Output "Uninstalled: $name (reboot required)"
            }
            return
        }

        # Quiet uninstall string (preferred — already includes silent flags)
        if ($entry.QuietUninstallString) {
            Write-Output "Uninstalling $name (quiet)"
            $code = Invoke-Uninstaller -CommandLine $entry.QuietUninstallString
            if ($code -notin $successCodes) {
                Write-Output "Failed: exit code $code"
                $failed = $true
            }
            return
        }

        # Standard uninstall string — caveat: may not support silent mode
        if ($entry.UninstallString) {
            Write-Output "Uninstalling $name (standard — may not be silent)"
            $code = Invoke-Uninstaller -CommandLine $entry.UninstallString -ExtraArgs "/quiet /norestart"
            if ($code -notin $successCodes) {
                Write-Output "Failed: exit code $code"
                $failed = $true
            }
            return
        }
    }
}

if (-not $found) {
    Write-Output "No McAfee software found to remove."
    exit 0
}

if ($failed) {
    Write-Output "One or more uninstallations failed. Check output above."
    exit 1
}

Write-Output "McAfee removal complete."
exit 0
```

This script reads both the 64-bit and 32-bit Uninstall registry hives, rebuilds MSI uninstall commands from the product GUID (handling both `/I` and `/X` registrations, quoted or unquoted msiexec paths), and parses EXE uninstall strings into executable-plus-arguments so paths with spaces — like anything under `Program Files` — launch correctly. It prefers `QuietUninstallString` when available and falls back to the standard `UninstallString` with appended quiet flags. Exit codes 3010 and 1641 (success, reboot required) are treated as success so a completed uninstall doesn't trigger a spurious retry. The script returns non-zero only on real failures, so Fleet's 3-retry mechanism triggers correctly.

4. Click **Save** to create the script.

> **Warning:** Test this on a single host first. Run the script manually against one machine and check the script output in that host's activity feed to confirm it uninstalls the right software before automating fleet-wide.

> **Note:** If your organization legitimately uses McAfee/Trellix Endpoint Security on some machines, narrow the scope. Replace `*McAfee*` with `*McAfee Trial*` or `*McAfee Safe Search*` to avoid removing production security software.

> **Caveat:** Appending `/quiet /norestart` to arbitrary EXE uninstallers doesn't always work — NSIS installers want `/S`, Inno Setup wants `/VERYSILENT`. If an uninstaller lacks quiet support, it will prompt for UI and hang in Fleet's non-interactive SYSTEM context until the script timeout. For stubborn software, use vendor-specific removal tools (e.g., McAfee's MCPR tool) deployed as a Fleet software package.

## Connect the policy and script with automation

1. Navigate to **Policies** and click **Manage automations**.
2. Find your "McAfee trial software detected" policy and select it.
3. In the automation panel, choose **Run script** and select "Remove McAfee trial software."
4. Click **Save**.

When any Windows host fails the McAfee policy, Fleet runs the uninstall script. The script runs up to 3 times total, retriggering each time it exits with a non-zero code. After removal, the next policy evaluation passes.

> **Note:** Policy automations attach to policies scoped to a specific fleet (team), not global policies. If you organize hosts by fleet, create the policy at that level and attach the script there.

## Detecting Store apps (MSIX) like the LG Monitor App

Companion apps like the LG Monitor App and Alienware Command Center install as MSIX packages from the Microsoft Store — no user action required. Since osquery 5.17.0, MSIX packages appear in the `programs` table with a populated `package_family_name` column, and osquery 5.22.1 closed the remaining gap where apps that no user had launched were missing from inventory. That means the same policy pattern works here — match on the package family name rather than the display name, since it's the stable identifier:

### Policy

1. Navigate to **Policies** and click **Add policy**.
2. In the **Name** field, enter "LG Monitor App (Store) detected."
3. In the **Query** field, paste:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM programs
  WHERE package_family_name LIKE 'LGElectronics.LGMonitorApp%'
);
```

This returns zero rows (fail) when the LG Monitor App package is present on the host.

> **Note:** To find the package family name for any Store app, run `Get-AppxPackage -AllUsers | Select Name, PackageFamilyName` on an affected host, or query `SELECT name, package_family_name FROM programs WHERE package_family_name != ''` via Fleet live query. Vendors sometimes ship a separate installer stub package alongside the app itself — check for related packages (for example, names containing "Installer") and widen the `LIKE` pattern if you find one.

> **Note:** Querying MSIX data in `programs` involves enumerating installed packages through the Windows Appx APIs, which is slower than the registry reads used for MSI/EXE entries. Policy evaluations run on a schedule (default hourly), so this doesn't affect end users, but live queries can take longer on hosts with many Store apps.

### Removal script

1. Navigate to **Controls > Scripts** and click **Add script**.
2. Name the script "Remove LG Monitor App and prevent reinstall" and set **Platform** to Windows.
3. In the **Script** field, paste:

```powershell
$failed = $false

# Remove the LG Monitor App (Store/MSIX package) for all users.
# Wildcard also catches related packages (e.g. installer stubs).
$packages = Get-AppxPackage -Name "LGElectronics.LGMonitorApp*" -AllUsers -ErrorAction SilentlyContinue

if ($packages) {
    foreach ($package in $packages) {
        Write-Output "Removing $($package.Name) version $($package.Version)"
        try {
            $package | Remove-AppxPackage -AllUsers -ErrorAction Stop
            Write-Output "App removed."
        } catch {
            Write-Output "App removal failed: $($_.Exception.Message)"
            $failed = $true
        }
    }
} else {
    Write-Output "LG Monitor App not found (may already be removed)."
}

# Remove the provisioned package so it isn't installed for new users
try {
    $provisioned = Get-AppxProvisionedPackage -Online -ErrorAction Stop |
        Where-Object { $_.DisplayName -like "*LGMonitorApp*" }
} catch {
    $provisioned = $null
}

if ($provisioned) {
    try {
        $provisioned | Remove-AppxProvisionedPackage -Online -ErrorAction Stop | Out-Null
        Write-Output "Provisioned package removed."
    } catch {
        Write-Output "Provisioned package removal failed: $($_.Exception.Message)"
        $failed = $true
    }
}

# Remove LG's driver-store delivery packages. LG ships SoftwareComponent
# driver packages matched to monitor hardware IDs whose job is to re-trigger
# the Store install. They survive app removal and re-arm the install cycle,
# so removing the app alone is not enough. Removing them does not affect
# basic monitor functionality.
try {
    $lgDrivers = Get-WindowsDriver -Online -ErrorAction Stop |
        Where-Object {
            $_.ProviderName -like "LG Electronics*" -and
            $_.ClassName -in @("SoftwareComponent", "Extension")
        }
} catch {
    $lgDrivers = @()
    Write-Output "Could not enumerate the driver store: $($_.Exception.Message)"
}

foreach ($drv in $lgDrivers) {
    Write-Output "Removing driver package $($drv.Driver) ($($drv.OriginalFileName))"
    $null = pnputil /delete-driver $drv.Driver /uninstall /force
    if ($LASTEXITCODE -notin @(0, 3010)) {
        Write-Output "Failed to remove $($drv.Driver): pnputil exit code $LASTEXITCODE"
        $failed = $true
    }
}

# Belt and suspenders: block device metadata retrieval, which is one of the
# channels Windows uses to deliver companion apps for connected hardware.
# Note: this does NOT block installs triggered by driver-store packages —
# that's what the pnputil cleanup above is for.
$policyPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows\Device Metadata"
if (-not (Test-Path $policyPath)) {
    New-Item -Path $policyPath -Force | Out-Null
}
$existing = (Get-ItemProperty -Path $policyPath -Name "PreventDeviceMetadataFromNetwork" -ErrorAction SilentlyContinue).PreventDeviceMetadataFromNetwork
if ($existing -eq 1) {
    Write-Output "Device metadata retrieval already disabled."
} else {
    Set-ItemProperty -Path $policyPath -Name "PreventDeviceMetadataFromNetwork" -Value 1 -Type DWord
    Write-Output "Device metadata retrieval disabled."
}

if ($failed) { exit 1 }
exit 0
```

4. Save the script and attach it to the Store app policy via **Manage automations**.

> **Warning:** Before deploying, run `pnputil /enum-drivers` on an affected host and confirm the LG delivery packages' provider name and class. Adjust the `ProviderName` filter if your hosts report a different string. Only `SoftwareComponent` and `Extension` class packages are targeted — display drivers are untouched.

> **Warning:** Disabling device metadata retrieval blocks companion app installation via device metadata for ALL hardware — including legitimate ones your users may want. Scope this to specific fleets rather than applying it globally.

> **Note:** `PreventDeviceMetadataFromNetwork` is also settable through Windows MDM as an ADMX-backed policy (`./Device/Vendor/MSFT/Policy/Config/DeviceInstallation/PreventDeviceMetadataFromNetwork`). If you manage Windows hosts with Fleet MDM, a custom configuration profile is the more durable option — profiles are re-enforced, while a script sets the value once. The script approach above works on hosts without MDM enrollment.

## Get notified when unwanted software is detected

Fleet sends webhook notifications when a host transitions to a failing policy state. Webhooks fire once per day by default, not immediately. To send Slack notifications, you need a transform layer because Fleet's JSON payload doesn't match Slack's expected `{"text": "..."}` format:

1. Set up an incoming webhook in Tines, Zapier, or a Lambda function that transforms Fleet's payload into Slack format.
2. Navigate to **Policies > Manage automations**, enable the webhook workflow, select your policy, and enter your transform layer's URL.

> **Note:** Webhook notifications are available on Fleet Free. Script automations require Fleet Premium.

## Adapt this for other peripheral-installed software

The same pattern works for any unwanted software:

- **Docking station utilities.** DisplayLink Manager, Plugable utilities, and other dock software show up in `programs`. Use the policy template and scope by `publisher` plus a specific product name.
- **Printer bundles.** Canon, Epson, and Brother utilities follow the same approach. Scope by publisher to avoid hitting unrelated software.
- **Monitor companion apps (Store).** Alienware Command Center auto-installs via the same mechanisms. Use the same `package_family_name LIKE '...'` pattern — run `Get-AppxPackage -AllUsers | Select PackageFamilyName` on an affected host to get the exact prefix.

> **Note:** Avoid broad substring matches like `%HP%` in the `name` field — they hit unrelated programs. Scope on `publisher` or use specific product names.

## Verify the cleanup worked

1. Navigate to **Software** and search for "McAfee" in the software inventory.
2. Confirm the number of affected hosts drops to zero as policies evaluate.

You can also run a live query from **Queries**:

```sql
SELECT name, version, publisher, install_date FROM programs WHERE name LIKE '%McAfee%';
```

If no results return, the software is fully removed from your fleet.

## Troubleshoot

**Policy automation didn't trigger for hosts that were already failing.**

Automations fire on transition (newly failing: no-response-to-fail or pass-to-fail). Hosts that were already failing won't trigger. To force a recheck: deselect the policy in **Manage automations**, click **Save**, then reselect it. This resets the host counts and re-triggers the automation immediately.

**Store app doesn't appear in the software inventory.**

Check the host's osquery version (`SELECT version FROM osquery_info;`). MSIX support in the `programs` table requires osquery 5.17.0, and apps that no user has launched — the normal state for auto-installed companion apps — require 5.22.1. On older agents, update the fleetd package to bring Store apps into inventory.

**Script hangs or times out on some hosts.**

If an uninstaller lacks quiet/silent flags, it may prompt for UI input — which fails in Fleet's non-interactive SYSTEM context and hangs until the script timeout. The timeout is an agent option (`script_execution_timeout` under `agent_options`, default 300 seconds, maximum 18000), settable through the UI or GitOps. For stubborn software, use vendor-specific removal tools deployed as Fleet software packages.

**Store app keeps reinstalling after removal.**

Windows has two delivery channels that can re-trigger the install when the user reconnects the peripheral. The first is device metadata: Windows matches the hardware to a companion app listing and installs it. The `PreventDeviceMetadataFromNetwork` policy blocks this channel. The second is driver-store delivery: the vendor ships a `SoftwareComponent` driver package (via Windows Update, matched to hardware IDs) whose only job is to install the Store app. The metadata policy does NOT block this channel — the driver package must be removed from the driver store with `pnputil`, which the removal script above does. If the app still returns, check `pnputil /enum-drivers` output for vendor packages the script's filter missed, and check whether Windows Update re-delivered the driver package (block it with a driver group policy or WSUS/WUfB deferral if so).

**Automation retry limit reached.**

Script automations attempt up to 3 times, retriggering on non-zero exit codes. If all 3 fail, Fleet stops retrying. Check the script output in the host's activity feed to see why it failed. In Fleet Premium, set `continuous_automations_enabled: true` on the policy to trigger on every evaluation, including fail-to-fail transitions.

## Further reading

- [Policy automations](https://fleetdm.com/guides/automations) — Configure webhooks and script triggers for policies.
- [Run scripts on policy failure](https://fleetdm.com/guides/policy-automation-run-script) — Step-by-step for connecting policies to remediation scripts.
- [Provisioned MSIX apps in software inventory (fleetdm/fleet#39065)](https://github.com/fleetdm/fleet/issues/39065) — Background on the osquery 5.22.1 fix that makes never-launched Store apps visible in `programs`.

<meta name="articleTitle" value="Detect and remove unwanted software installed by peripherals in Fleet">
<meta name="authorFullName" value="Dhruv Majumdar">
<meta name="authorGitHubUsername" value="dmajumdar">
<meta name="publishedOn" value="2026-07-21">
<meta name="category" value="guides">
<meta name="description" value="Detect and remove unwanted software installed by peripherals and docking stations using Fleet policies and scripts.">
