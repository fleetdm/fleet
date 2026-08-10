# Add Microsoft Store apps to Windows self-service with winget

Fleet 4.89.0 gave script-only packages an uninstall script, a pre-install query, and a post-install script. That is enough to put Microsoft Store apps on your end users' self-service page using winget, with no packages to host. This guide builds one Store app tile end to end, then generates the rest from a Store ID. It covers per-user Store installs driven through self-service, which is the only scope the Store supports through winget.

> **Warning:** `winget install --source msstore` cannot run as SYSTEM, and Fleet runs Windows scripts as SYSTEM. The Store rejects device-wide installs with "Device wide install for msstore type is not supported under admin context." Step 3 works around this by running winget in the logged-on user's session. If you need a Store app installed machine-wide for every user, see [Deploy a Store app machine-wide](#deploy-a-store-app-machine-wide) instead.

## Prerequisites

- Fleet 4.89.0 or later. The uninstall script, pre-install query, and post-install script on script-only packages were added in this release.
- A GitOps repository connected to your Fleet instance. Software is defined per fleet in `fleets/<name>.yml`.
- Windows hosts enrolled in Fleet with scripts enabled. See the [scripts guide](https://fleetdm.com/guides/scripts) if you deployed Fleet's agent without `--enable-scripts`.
- App Installer present on those hosts, which provides winget. Windows Server and LTSC images often ship without it.
- Fleet Desktop available to end users, so the self-service page is reachable from the Windows system tray.

> **Note:** Script-only packages do not support an `install_script` key, because the file's contents already are the install script. They also do not support automatic install through a policy. Drive these through self-service.

## Step 1: Find the Store ID and package name

You need two identifiers, and they are not the same thing.

1. Get the **Store ID**, a twelve-character string winget uses to install the app.

   ```powershell
   winget search --source msstore "company portal"
   ```

   For Company Portal, the ID is `9WZDNCRFJ3PZ`.

2. Install the app by hand on one machine, then get the **MSIX package name**, which you need for verification in Step 5.

   ```powershell
   Get-AppxPackage | Select-Object Name, PackageFamilyName
   ```

   For Company Portal, the name is `Microsoft.CompanyPortal`.

> **Note:** Always pin to the Store ID rather than the app name. A name match can resolve to the wrong package, and nothing is watching the output when Fleet runs the script.

## Step 2: Confirm the commands you are wrapping

These are the only two winget commands involved. Everything in Step 3 exists to get them running in the right place.

```powershell
winget install --id 9WZDNCRFJ3PZ --source msstore --accept-package-agreements --accept-source-agreements --disable-interactivity
winget uninstall --id 9WZDNCRFJ3PZ --accept-source-agreements --disable-interactivity
```

> **Warning:** Keep the agreement flags on the uninstall too. winget [prompts for msstore source agreements even on uninstall](https://github.com/microsoft/winget-cli/issues/1736), and in an unattended script that prompt hangs forever. `--disable-interactivity` turns any remaining prompt into a failure instead.

## Step 3: Write the install script

Fleet runs the script as SYSTEM, where neither winget nor the Store will cooperate. Create a short-lived scheduled task owned by the logged-on user, start it, wait for it to finish, then remove it.

Save this as `install-company-portal.ps1`.

```powershell
$StoreId = "9WZDNCRFJ3PZ"
$WingetArgs = "install --id $StoreId --source msstore --accept-package-agreements --accept-source-agreements --disable-interactivity"

$exitCode = 0
$taskName = "fleet-store-$StoreId"

try {
    $userName = (Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' |
        Invoke-CimMethod -MethodName GetOwner | Select-Object -First 1).User
    if (-not $userName) { throw "No logged-on user, so there is no session to install into." }

    $action = New-ScheduledTaskAction -Execute "winget.exe" -Argument $WingetArgs
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
            throw "Timed out waiting for winget to finish."
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

Only the first two lines are app-specific. Inside the user's session, plain `winget.exe` resolves through the app execution alias, so no path resolution is needed.

> **Warning:** The scheduled task's exit code does not come back to this script, so it reports success whenever the task ran, whether or not winget installed anything. Step 5 is what catches that, and it is not optional here.

## Step 4: Write the uninstall script

Copy the install script and change one line. Save it as `uninstall-company-portal.ps1`.

```powershell
$WingetArgs = "uninstall --id $StoreId --accept-source-agreements --disable-interactivity"
```

> **Note:** If winget cannot match the installed package on uninstall, remove it directly with `Remove-AppxPackage` inside the same scheduled task, using the package family name from Step 1.

## Step 5: Verify the install landed

Store apps do not register in `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, so a registry check fails on a good install. Ask the MSIX subsystem instead. Fleet runs this as SYSTEM, which is elevated, so `-AllUsers` will see a package installed in a user's profile.

Save this as `verify-company-portal.ps1`.

```powershell
$PackageName = "Microsoft.CompanyPortal"

$pkg = Get-AppxPackage -AllUsers -Name $PackageName -ErrorAction SilentlyContinue
if (-not $pkg) {
    Write-Host "$PackageName is not installed for any user."
    exit 1
}

Write-Host "Found $($pkg[0].Name) $($pkg[0].Version)."
exit 0
```

A non-zero exit from the post-install script fails the install and triggers your uninstall script.

## Step 6: Register the package in your fleet file

Save all three scripts under `lib/windows/scripts/` in your GitOps repo. Then register the package in `fleets/<name>.yml` or `fleets/unassigned.yml`.

```yaml
labels:
  - name: Windows
    query: "SELECT 1 FROM os_version WHERE platform = 'windows';"
    label_membership_type: dynamic
```

```yaml
software:
  packages:
    - path: ../lib/windows/scripts/install-company-portal.ps1
      display_name: Company Portal
      self_service: true
      categories:
        - "💻 Productivity"
      labels_include_any:
        - Windows
      uninstall_script:
        path: ../lib/windows/scripts/uninstall-company-portal.ps1
      post_install_script:
        path: ../lib/windows/scripts/verify-company-portal.ps1
```

With `self_service: true`, the app appears on the end user's self-service page, reachable from the Fleet icon in the Windows system tray.

> **Note:** Any label you reference on a package must be defined in the `labels` section first. Use `labels_include_any`, `labels_include_all`, or `labels_exclude_any`, but only one per package. Use a label to exclude hosts without App Installer, such as Windows Server and LTSC images.

## Step 7: Generate new tiles from a Store ID

Only two lines differ between apps, so the rest can be emitted. Fleet script-only packages are single files with no shared helper to import, so the boilerplate is inlined into each generated script.

Save this as `New-StoreAppTile.ps1`.

```powershell
param(
    [Parameter(Mandatory)][string]$StoreId,
    [Parameter(Mandatory)][string]$DisplayName,
    [Parameter(Mandatory)][string]$PackageName,
    [string]$Category = "💻 Productivity"
)

$dir = "lib/windows/scripts"
New-Item -ItemType Directory -Path $dir -Force | Out-Null
$slug = $DisplayName.ToLower() -replace '[^a-z0-9]+', '-'

$body = @'
$exitCode = 0
$taskName = "fleet-store-$StoreId"
try {
    $userName = (Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' |
        Invoke-CimMethod -MethodName GetOwner | Select-Object -First 1).User
    if (-not $userName) { throw "No logged-on user, so there is no session to install into." }
    $action = New-ScheduledTaskAction -Execute "winget.exe" -Argument $WingetArgs
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
            throw "Timed out waiting for winget to finish."
        }
    } while ($state -eq "Running" -or $state -eq "Queued")
} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
}
exit $exitCode
'@

@"
`$StoreId = "$StoreId"
`$WingetArgs = "install --id `$StoreId --source msstore --accept-package-agreements --accept-source-agreements --disable-interactivity"
$body
"@ | Set-Content "$dir/install-$slug.ps1" -Encoding UTF8

@"
`$StoreId = "$StoreId"
`$WingetArgs = "uninstall --id `$StoreId --accept-source-agreements --disable-interactivity"
$body
"@ | Set-Content "$dir/uninstall-$slug.ps1" -Encoding UTF8

@"
`$PackageName = "$PackageName"
`$pkg = Get-AppxPackage -AllUsers -Name `$PackageName -ErrorAction SilentlyContinue
if (-not `$pkg) { Write-Host "`$PackageName is not installed for any user."; exit 1 }
Write-Host "Found `$(`$pkg[0].Name) `$(`$pkg[0].Version)."
exit 0
"@ | Set-Content "$dir/verify-$slug.ps1" -Encoding UTF8

@"

# Add to your fleet's software.packages:
    - path: ../$dir/install-$slug.ps1
      display_name: $DisplayName
      self_service: true
      categories:
        - "$Category"
      labels_include_any:
        - Windows
      uninstall_script:
        path: ../$dir/uninstall-$slug.ps1
      post_install_script:
        path: ../$dir/verify-$slug.ps1
"@
```

Run it with the two identifiers from Step 1, then paste the printed block into your fleet file and open a pull request.

```powershell
./New-StoreAppTile.ps1 -StoreId 9WZDNCRFJ3PZ -DisplayName "Company Portal" -PackageName Microsoft.CompanyPortal
```

## Deploy a Store app machine-wide

Self-service installs are per-user. If an app has to be present for every user on a host, provision it instead of installing it, which does work as SYSTEM.

1. On an admin workstation, download the package and its license.

   ```powershell
   winget download --id 9WZDNCRFJ3PZ --source msstore --download-directory C:\staging
   ```

2. Provision it on the host.

   ```powershell
   Add-AppxProvisionedPackage -Online -PackagePath .\app.msixbundle -LicensePath .\9WZDNCRFJ3PZ_License.xml
   ```

> **Warning:** Downloading a Store package's license file [requires Entra ID authentication](https://learn.microsoft.com/en-us/windows/package-manager/winget/download) by an account with the Global Administrator, User Administrator, or License Administrator role. You also now have a file to deliver, so use a Fleet custom package rather than a script-only one.

> **Note:** Do not reach for `winget install --scope machine` as a shortcut here. For a Store package it [installs under the SYSTEM account](https://github.com/microsoft/winget-cli/issues/4748) rather than provisioning the app, which looks like success and leaves no app for your users.

## Troubleshoot

**The install fails with "Device wide install for msstore type is not supported under admin context."** The script is running winget as SYSTEM. Store apps are per-user, so wrap the call in the scheduled task from Step 3, or provision the app machine-wide instead.

**The install reports success but the app is not there.** The scheduled task ran and winget failed inside it. The task's exit code never reaches your script, so add the verification script from Step 5, which fails the install and triggers the uninstall.

**Verification fails even though the app is installed.** Check that you are using `Get-AppxPackage -AllUsers` and the MSIX package name from Step 1, not the Store ID. Store apps never appear in the uninstall registry keys, so a registry-based check always fails here.

**Nothing happens and the script throws "No logged-on user."** There is no `explorer.exe` owner to run the task as. This is expected on a host at the login screen, and self-service installs assume somebody is signed in.

**The script fails because `winget` is not recognized.** The host has no App Installer package, which is common on Windows Server and LTSC images. Exclude those hosts with a label, or install App Installer first.

**The tile does not appear on a host's self-service page.** Confirm the host matches the package's label, and that the label query returns the host in the Fleet UI under the label's host list.

## Further reading

- [Deploy software guide](https://fleetdm.com/guides/deploy-software-packages) for full detail on script-only packages, pre-install queries, and uninstall scripts.
- [Put Microsoft Store apps in Windows self-service with winget](https://fleetdm.com/articles/build-your-own-windows-self-service-with-winget-and-script-only-packages) for the reasoning behind this approach.
- [winget troubleshooting](https://learn.microsoft.com/en-us/windows/package-manager/winget/troubleshooting) for Microsoft's guidance on the system context.
- [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files).

<meta name="articleTitle" value="Add Microsoft Store apps to Windows self-service with winget">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-07">
<meta name="description" value="Use Fleet script-only .ps1 packages and winget to add Microsoft Store apps to Windows self-service, per-user or machine-wide.">
