# Add Microsoft Store apps to Windows self-service with winget

You can add Microsoft Store apps to self-service using winget and script-only packages. Built-in support for Microsoft Store apps is [coming soon](https://github.com/fleetdm/fleet/issues/43493).


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

## Step 7: Generate script-only packages for more Store apps

Steps 3–6 give you three scripts and a YAML block for one app. For each additional Store app, only the Store ID and MSIX package name change, so you can generate these files instead of repeating Steps 3–6 by hand.

The helper script below runs on your own computer, from the root of your GitOps repo. It doesn't run on hosts, isn't uploaded to Fleet, and doesn't install anything. It:

- Writes the install, uninstall, and verify scripts from Steps 3–5 for the new app to `lib/windows/scripts/`.
- Prints the `software.packages` YAML from Step 6 for you to paste into your fleet file.

Its inputs are the Store ID and MSIX package name from Step 1, plus the display name end users see in self-service. Script-only packages are single files with no shared helper to import, so each generated script includes the full code from Step 3.

Save this as `New-StoreAppPackage.ps1` in the root of your GitOps repo.

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

Then add the app:

1. From the root of your GitOps repo, run the script with the identifiers from Step 1 and a display name.

   ```powershell
   ./New-StoreAppPackage.ps1 -StoreId 9WZDNCRFJ3PZ -DisplayName "Company Portal" -PackageName Microsoft.CompanyPortal
   ```

2. Paste the printed YAML under `software.packages` in `fleets/<name>.yml`.
3. Commit the three new scripts in `lib/windows/scripts/` and the change to your fleet file.
4. Open a pull request.

> **Note:** The printed paths, such as `../lib/windows/scripts/install-company-portal.ps1`, assume your fleet files are in `fleets/`, one level below the repo root. If yours are somewhere else, adjust the paths before you commit.

## Troubleshoot

**The install fails with "Device wide install for msstore type is not supported under admin context."** The script is running winget as SYSTEM. Store apps are per-user, so wrap the call in the scheduled task from Step 3, or provision the app machine-wide instead.

**The install reports success but the app is not there.** The scheduled task ran and winget failed inside it. The task's exit code never reaches your script, so add the verification script from Step 5, which fails the install and triggers the uninstall.

**Verification fails even though the app is installed.** Check that you are using `Get-AppxPackage -AllUsers` and the MSIX package name from Step 1, not the Store ID. Store apps never appear in the uninstall registry keys, so a registry-based check always fails here.

**Nothing happens and the script throws "No logged-on user."** There is no `explorer.exe` owner to run the task as. This is expected on a host at the login screen, and self-service installs assume somebody is signed in.

**The script fails because `winget` is not recognized.** The host has no App Installer package, which is common on Windows Server and LTSC images. Exclude those hosts with a label, or install App Installer first.

**The software does not appear on a host's self-service page.** Confirm the host matches the package's label, and that the label query returns the host in the Fleet UI under the label's host list.

## Advanced

### Deploy a Store app machine-wide

Self-service installs are per-user. If an app has to be present for every user on a host, provision it instead of installing it, which does work as SYSTEM. Fleet doesn't accept `.msix`, `.msixbundle`, or `.zip` files as custom packages, so you host the downloaded files yourself and a script-only package fetches them.

1. On an admin workstation, download the package, its dependencies, and its license.

   ```powershell
   winget download --id 9WZDNCRFJ3PZ --source msstore --download-directory C:\staging
   ```

2. Zip the downloaded files.

   ```powershell
   Compress-Archive -Path C:\staging\* -DestinationPath C:\company-portal.zip
   ```

3. Upload `company-portal.zip` to a location your hosts can download from over HTTPS, such as an internal web server or a cloud storage bucket.

4. Save this install script as `provision-company-portal.ps1`. Set `$PackageUrl` to the URL from the previous step.

   ```powershell
   $PackageUrl = "https://example.com/company-portal.zip"

   $staging = Join-Path $env:TEMP "fleet-store-9WZDNCRFJ3PZ"
   $zip = "$staging.zip"

   try {
       Invoke-WebRequest -Uri $PackageUrl -OutFile $zip -UseBasicParsing
       Expand-Archive -Path $zip -DestinationPath $staging -Force

       $package = Get-ChildItem -Path "$staging\*" -File -Include *.msixbundle, *.appxbundle, *.msix, *.appx | Select-Object -First 1
       $license = Get-ChildItem $staging -Recurse -Filter *_License.xml | Select-Object -First 1
       $dependencies = Get-ChildItem $staging -Recurse -File -Include *.msix, *.appx |
           Where-Object { $_.DirectoryName -ne $staging } | Select-Object -ExpandProperty FullName

       $params = @{ Online = $true; PackagePath = $package.FullName; LicensePath = $license.FullName }
       if ($dependencies) { $params.DependencyPackagePath = $dependencies }
       Add-AppxProvisionedPackage @params
   } catch {
       Write-Host "Error: $_"
       exit 1
   } finally {
       Remove-Item $staging, $zip -Recurse -Force -ErrorAction SilentlyContinue
   }
   ```

5. Save this uninstall script as `deprovision-company-portal.ps1`. It removes the provisioned package and any per-user copies, using the MSIX package name from Step 1.

   ```powershell
   $PackageName = "Microsoft.CompanyPortal"
   Get-AppxProvisionedPackage -Online | Where-Object DisplayName -eq $PackageName | Remove-AppxProvisionedPackage -Online
   Get-AppxPackage -AllUsers -Name $PackageName | Remove-AppxPackage -AllUsers
   ```

6. In Fleet, go to **Software > Add software > Custom package** and upload `provision-company-portal.ps1`. Open **Advanced options** and paste the contents of `deprovision-company-portal.ps1` as the uninstall script.

> **Warning:** Downloading a Store package's license file [requires Entra ID authentication](https://learn.microsoft.com/en-us/windows/package-manager/winget/download) by an account with the Global Administrator, User Administrator, or License Administrator role.

> **Note:** Do not reach for `winget install --scope machine` as a shortcut here. For a Store package it [installs under the SYSTEM account](https://github.com/microsoft/winget-cli/issues/4748) rather than provisioning the app, which looks like success and leaves no app for your users.

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
