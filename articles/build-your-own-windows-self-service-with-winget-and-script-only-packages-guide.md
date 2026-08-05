# Build a Windows self-service catalog with winget

Fleet 4.89.0 gave script-only packages an uninstall script, a pre-install query, and a post-install script. That is enough to turn winget into a self-service software catalog for Windows, defined entirely in Git, with no `.msi` or `.exe` files to build or host. This guide walks through building one app end to end, then generating the rest from a winget package ID. It covers machine-scope installs from the community `winget` source, plus an optional path for user-scope apps.

> **Warning:** Fleet runs Windows scripts as SYSTEM, and Microsoft [does not support the winget CLI in the system context](https://learn.microsoft.com/en-us/windows/package-manager/winget/troubleshooting). Step 1 works around that for machine-scope installs. Microsoft Store apps from the `msstore` source cannot be installed device-wide at all, so those need Step 7.

## Prerequisites

- Fleet 4.89.0 or later. The uninstall script, pre-install query, and post-install script on script-only packages were added in this release.
- A GitOps repository connected to your Fleet instance. Software is defined per fleet in `fleets/<name>.yml`.
- Windows hosts enrolled in Fleet with scripts enabled. See the [scripts guide](https://fleetdm.com/guides/scripts) if you deployed Fleet's agent without `--enable-scripts`.
- App Installer present on those hosts, which provides winget. Windows Server and LTSC images often ship without it.
- Fleet Desktop available to end users, so the self-service page is reachable from the Windows system tray.

> **Note:** Script-only packages do not support an `install_script` key, because the file's contents already are the install script. They also do not support automatic install through a policy. Drive installs through self-service.

## Step 1: Locate winget in the SYSTEM context

The `winget` app execution alias lives in a per-user path, so SYSTEM has no `winget` command. The binary is still on disk inside the App Installer package directory. Resolve it there.

The directory name embeds the version, so sort on the parsed version rather than the string. A plain descending sort puts `1.9` above `1.24`.

```powershell
$winget = Resolve-Path "$env:ProgramFiles\WindowsApps\Microsoft.DesktopAppInstaller_*_x64__8wekyb3d8bbwe\winget.exe" -ErrorAction SilentlyContinue |
    Sort-Object { [version](($_.Path -split '_')[1]) } |
    Select-Object -Last 1 -ExpandProperty Path

if (-not $winget) {
    Write-Host "App Installer (winget) was not found on this host."
    exit 1
}
```

Both scripts below open with this block.

## Step 2: Write the install script

Save this as `install-7zip.ps1`.

```powershell
$PackageId = "7zip.7zip"

$winget = Resolve-Path "$env:ProgramFiles\WindowsApps\Microsoft.DesktopAppInstaller_*_x64__8wekyb3d8bbwe\winget.exe" -ErrorAction SilentlyContinue |
    Sort-Object { [version](($_.Path -split '_')[1]) } |
    Select-Object -Last 1 -ExpandProperty Path

if (-not $winget) {
    Write-Host "App Installer (winget) was not found on this host."
    exit 1
}

Write-Host "Installing $PackageId with $winget"
& $winget install --id $PackageId --exact --source winget `
    --scope machine --silent `
    --accept-package-agreements --accept-source-agreements `
    --disable-interactivity
$code = $LASTEXITCODE

# 0 = installed. -1978335135 (0x8A150061) = already installed, which is not a failure.
if ($code -eq 0 -or $code -eq -1978335135) {
    Write-Host "$PackageId is installed."
    exit 0
}

Write-Host "winget exited with $code"
exit 1
```

Use `--exact` with `--id` so winget cannot resolve a fuzzy name match to the wrong package. The `--silent`, `--accept-package-agreements`, `--accept-source-agreements`, and `--disable-interactivity` flags keep the run unattended.

> **Warning:** PowerShell ignores a native command's non-zero exit code. `$ErrorActionPreference` governs PowerShell errors, not process exit codes, so a failed `winget install` leaves your script exiting 0 and Fleet marking the install successful. Check `$LASTEXITCODE` and propagate it, as above.

> **Note:** Microsoft documents that `--scope machine` is reliable for MSI and MSIX packages but not deterministic for EXE-based installers, where the scope arguments may not exist at all. Verify per app.

## Step 3: Write the uninstall script

Save this as `uninstall-7zip.ps1`. Removing something already absent is a successful no-op, so treat `-1978335212` as success.

```powershell
$PackageId = "7zip.7zip"

$winget = Resolve-Path "$env:ProgramFiles\WindowsApps\Microsoft.DesktopAppInstaller_*_x64__8wekyb3d8bbwe\winget.exe" -ErrorAction SilentlyContinue |
    Sort-Object { [version](($_.Path -split '_')[1]) } |
    Select-Object -Last 1 -ExpandProperty Path

if (-not $winget) {
    Write-Host "App Installer (winget) was not found on this host."
    exit 1
}

& $winget uninstall --id $PackageId --exact --silent `
    --accept-source-agreements --disable-interactivity
$code = $LASTEXITCODE

# -1978335212 (0x8A150014) = no packages found, so there is nothing to remove.
if ($code -eq 0 -or $code -eq -1978335212) {
    Write-Host "$PackageId is not installed."
    exit 0
}

Write-Host "winget exited with $code"
exit 1
```

## Step 4: Register the package in your fleet file

Save both scripts under `lib/windows/scripts/` in your GitOps repo. Then register the package in the fleet where you want it available, in `fleets/<name>.yml` or `fleets/unassigned.yml`.

```yaml
software:
  packages:
    - path: ../lib/windows/scripts/install-7zip.ps1
      display_name: 7-Zip
      self_service: true
      categories:
        - "💻 Productivity"
      uninstall_script:
        path: ../lib/windows/scripts/uninstall-7zip.ps1
```

With `self_service: true`, the app appears on the end user's self-service page, reachable from the Fleet icon in the Windows system tray. When the user clicks install, Fleet's agent runs the install script as SYSTEM. When they remove it, the uninstall script runs.

## Step 5: Target the right hosts with labels

Define a dynamic label, then reference it on the package so only the intended hosts see the tile.

```yaml
labels:
  - name: Windows
    query: "SELECT 1 FROM os_version WHERE platform = 'windows';"
    label_membership_type: dynamic
```

```yaml
software:
  packages:
    - path: ../lib/windows/scripts/install-7zip.ps1
      display_name: 7-Zip
      self_service: true
      labels_include_any:
        - Windows
      uninstall_script:
        path: ../lib/windows/scripts/uninstall-7zip.ps1
```

Use labels to exclude the hosts winget cannot help, such as Windows Server and LTSC images without App Installer.

> **Note:** Any label you reference on a package must be defined in the `labels` section first. Use `labels_include_any`, `labels_include_all`, or `labels_exclude_any`, but only one per package.

## Step 6: Verify the install landed

A post-install script runs after the install. A non-zero exit fails the install and triggers your uninstall script. Check the state of the machine rather than asking winget again, so a broken winget cannot vouch for itself.

Save this as `verify-7zip.ps1`.

```powershell
$DisplayNamePattern = "7-Zip*"

$paths = @(
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*",
    "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*"
)

$found = Get-ItemProperty -Path $paths -ErrorAction SilentlyContinue |
    Where-Object { $_.DisplayName -like $DisplayNamePattern }

if (-not $found) {
    Write-Host "$DisplayNamePattern was not found in the uninstall registry."
    exit 1
}

Write-Host "Found $($found[0].DisplayName) $($found[0].DisplayVersion)."
exit 0
```

Reference it on the package:

```yaml
      post_install_script:
        path: ../lib/windows/scripts/verify-7zip.ps1
```

> **Note:** For a user-scope install, `HKLM` is the wrong place to look. Those entries land under the user's own hive, so enumerate `HKEY_USERS` from SYSTEM. `HKCU` as SYSTEM points at the wrong profile.

## Step 7: Install user-scope apps as the logged-on user (optional)

Some apps refuse to install machine-wide. Squirrel-based Electron apps, anything from the Microsoft Store, and a long tail of `--scope user` packages all need a user profile to install into, and no SYSTEM-context workaround changes that.

Run winget as the user instead. Create a short-lived scheduled task that runs as the owner of the `explorer.exe` process, start it, wait for it to finish, then unregister it. This is the same pattern Fleet uses for its own per-user Windows [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps).

```powershell
$PackageId = "Figma.Figma"
$exitCode = 0
$taskName = "fleet-install-$PackageId"

try {
    $userName = (Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' |
        Invoke-CimMethod -MethodName GetOwner).User
    if (-not $userName) { throw "No logged-on user found; cannot install a user-scope app." }

    $args = "install --id $PackageId --exact --source winget --scope user --silent " +
            "--accept-package-agreements --accept-source-agreements --disable-interactivity"
    $action = New-ScheduledTaskAction -Execute "winget.exe" -Argument $args
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries
    $task = New-ScheduledTask -Action $action -Settings $settings

    Register-ScheduledTask $taskName -InputObject $task -User $userName | Out-Null
    Start-ScheduledTask -TaskName $taskName -TaskPath "\"

    $startDate = Get-Date
    do {
        Start-Sleep -Seconds 5
        $state = (Get-ScheduledTask -TaskName $taskName).State
        Write-Host "Scheduled task is '$state'."
        if ((New-TimeSpan -Start $startDate -End (Get-Date)).TotalSeconds -gt 600) {
            throw "Timed out waiting for the install to finish."
        }
    } while ($state -eq "Running" -or $state -eq "Queued")
} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
}

exit $exitCode
```

Because the task runs in the user's session, plain `winget.exe` resolves through the app execution alias and no path hunting is needed. Swap `install` for `uninstall` to build the matching removal script.

> **Warning:** The scheduled task's own exit code does not come back to your script, so this path reports success even when winget failed. The verification script in Step 6 is mandatory here, reading `HKEY_USERS` rather than `HKLM`.

## Step 8: Generate new apps from a winget ID

Once the pattern is settled, the only real input is the package ID. This generator writes both scripts and prints the YAML block to paste into your fleet file. Save it as `New-SelfServiceApp.ps1`.

```powershell
param(
    [Parameter(Mandatory)][string]$PackageId,
    [string]$DisplayName = $PackageId,
    [string]$Category = "💻 Productivity"
)

$dir = "lib/windows/scripts"
New-Item -ItemType Directory -Path $dir -Force | Out-Null
$slug = $PackageId.ToLower().Replace(".", "-")

$resolve = @'
$winget = Resolve-Path "$env:ProgramFiles\WindowsApps\Microsoft.DesktopAppInstaller_*_x64__8wekyb3d8bbwe\winget.exe" -ErrorAction SilentlyContinue |
    Sort-Object { [version](($_.Path -split '_')[1]) } |
    Select-Object -Last 1 -ExpandProperty Path
if (-not $winget) { Write-Host "App Installer (winget) was not found."; exit 1 }
'@

@"
`$PackageId = "$PackageId"
$resolve
& `$winget install --id `$PackageId --exact --source winget --scope machine --silent ``
    --accept-package-agreements --accept-source-agreements --disable-interactivity
`$code = `$LASTEXITCODE
if (`$code -eq 0 -or `$code -eq -1978335135) { exit 0 }
Write-Host "winget exited with `$code"
exit 1
"@ | Set-Content "$dir/install-$slug.ps1" -Encoding UTF8

@"
`$PackageId = "$PackageId"
$resolve
& `$winget uninstall --id `$PackageId --exact --silent ``
    --accept-source-agreements --disable-interactivity
`$code = `$LASTEXITCODE
if (`$code -eq 0 -or `$code -eq -1978335212) { exit 0 }
Write-Host "winget exited with `$code"
exit 1
"@ | Set-Content "$dir/uninstall-$slug.ps1" -Encoding UTF8

@"

# Add to your fleet's software.packages:
    - path: ../$dir/install-$slug.ps1
      display_name: $DisplayName
      self_service: true
      categories:
        - "$Category"
      uninstall_script:
        path: ../$dir/uninstall-$slug.ps1
"@
```

To onboard an app, find the ID with `winget search`, run the generator, paste the printed block into your fleet file, and open a pull request.

```powershell
./New-SelfServiceApp.ps1 -PackageId 7zip.7zip -DisplayName "7-Zip"
```

Write the verification script from Step 6 by hand. The display name in the registry rarely matches the winget ID closely enough to generate.

## Troubleshoot

**The install reports success but the app is not present.** Add the post-install verification script from Step 6. Without it, a script that swallows winget's exit code leaves the tile claiming success. This is the default failure mode on Windows, because PowerShell does not stop on a non-zero native exit code.

**The script fails with "App Installer (winget) was not found on this host."** The host has no App Installer package, which is common on Windows Server and LTSC images. Exclude those hosts with a label, or install App Installer first.

**The install fails with "Device wide install for msstore type is not supported under admin context."** The package is a Microsoft Store app, and Store packages cannot be installed device-wide. Use the community `winget` source if the app is published there, or follow Step 7.

**The install hangs.** A prompt is waiting for input that will never come. Confirm `--silent`, `--accept-package-agreements`, `--accept-source-agreements`, and `--disable-interactivity` are all on the command.

**The tile does not appear on a host's self-service page.** Check that the host matches the package's label. Confirm the label query returns the host in the Fleet UI under the label's host list.

**A user-scope install does nothing.** No one is logged in, so there is no `explorer.exe` owner to run the task as. The script in Step 7 throws in that case rather than reporting success.

## Further reading

- [Deploy software guide](https://fleetdm.com/guides/deploy-software-packages) for full detail on script-only packages, pre-install queries, and uninstall scripts.
- [Build your own Windows self-service with winget](https://fleetdm.com/articles/build-your-own-windows-self-service-with-winget-and-script-only-packages) for the reasoning behind this approach.
- [winget troubleshooting](https://learn.microsoft.com/en-us/windows/package-manager/winget/troubleshooting) for Microsoft's guidance on the system context and installer scope.
- [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files).

<meta name="articleTitle" value="Build a Windows self-service catalog with winget">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-05">
<meta name="description" value="Use Fleet script-only .ps1 packages and winget to build a Windows self-service catalog, including the SYSTEM context workaround.">
