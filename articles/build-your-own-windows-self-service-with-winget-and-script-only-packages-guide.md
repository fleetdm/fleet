# Add Microsoft Store apps to Windows with script-only packages

You can install Microsoft Store apps on Windows hosts, for all users, with a script-only package. The install script runs as SYSTEM and calls the Microsoft Store install service, the same one the `winget` CLI uses, so no user needs to be signed in. Built-in support for Microsoft Store apps is [coming soon](https://github.com/fleetdm/fleet/issues/43493).

## Prerequisites

- Fleet 4.89.0 or later. The uninstall script, pre-install query, and post-install script on script-only packages were added in this release.
- Windows hosts enrolled in Fleet with scripts enabled. See the [scripts guide](https://fleetdm.com/guides/scripts) if you deployed Fleet's agent without `--enable-scripts`.
- A free Microsoft Store app. Paid apps fail to install with a licensing error.

> **Note:** The Microsoft Store install service is documented for Microsoft's own apps, such as winget, and isn't officially supported for third parties. Microsoft could change it.

## Step 1: Find the Store ID and package family name

You need two identifiers.

1. Find the app on the [Microsoft Store website](https://apps.microsoft.com). The **Store ID** is the last part of the URL. For Company Portal (`https://apps.microsoft.com/detail/9WZDNCRFJ3PZ`), it's `9WZDNCRFJ3PZ`.

2. Get the **package family name** and confirm the app is supported. Run this on any computer, replacing the Store ID:

   ```powershell
   (Invoke-RestMethod "https://storeedgefd.dsx.mp.microsoft.com/v9.0/packageManifests/9WZDNCRFJ3PZ?Market=US").Data.Versions.Installers |
       Select-Object InstallerType, PackageFamilyName -Unique
   ```

   For Company Portal, `InstallerType` is `msstore` and `PackageFamilyName` is `Microsoft.CompanyPortal_8wekyb3d8bbwe`.

> **Note:** The scripts in this guide are written for MSIX apps: `InstallerType` is `msstore`, and the Store ID starts with `9`. Apps that use their own installer (MSI or EXE) have Store IDs that start with `XP`. They have no package family name, and their installer decides whether to install for one user or all users, so the scripts' all-users check fails for them. You can adapt the scripts, or add these apps as a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps) or [custom package](https://fleetdm.com/guides/deploy-software-packages).

## Step 2: Write the install script

Save this as `install-company-portal.ps1`. Only the first two variables are app-specific. Set them to the identifiers from Step 1.

```powershell
# Installs a Microsoft Store app for all users on the host.
# If the app is already installed, the Microsoft Store installs the latest version.

# Change these two lines for each app (see Step 1).
$storeId           = '9WZDNCRFJ3PZ'
$packageFamilyName = 'Microsoft.CompanyPortal_8wekyb3d8bbwe'
$packageName       = $packageFamilyName.Split('_')[0]
$timeoutMinutes    = 4   # Fleet stops scripts after 5 minutes by default

# Store errors this script explains:
$STORE_LICENSE_FAILED = -2143322111   # The Microsoft Store couldn't license the app (0x803F8001)
$PACKAGES_IN_USE      = -2147009278   # The app is open and couldn't be closed (0x80073D02)

Add-Type -AssemblyName System.Runtime.WindowsRuntime
$null = [Windows.ApplicationModel.Store.Preview.InstallControl.AppInstallManager, Windows.ApplicationModel.Store.Preview, ContentType = WindowsRuntime]
$null = [Windows.ApplicationModel.Store.Preview.InstallControl.AppInstallOptions, Windows.ApplicationModel.Store.Preview, ContentType = WindowsRuntime]
$null = [Windows.ApplicationModel.Store.Preview.InstallControl.AppInstallItem, Windows.ApplicationModel.Store.Preview, ContentType = WindowsRuntime]
$null = [Windows.ApplicationModel.Store.Preview.InstallControl.GetEntitlementResult, Windows.ApplicationModel.Store.Preview, ContentType = WindowsRuntime]
$null = [Windows.Management.Deployment.PackageManager, Windows.Management.Deployment, ContentType = WindowsRuntime]
$null = [Windows.Management.Deployment.DeploymentResult, Windows.Management.Deployment, ContentType = WindowsRuntime]
$null = [Windows.Management.Deployment.DeploymentProgress, Windows.Management.Deployment, ContentType = WindowsRuntime]

# Windows PowerShell can't await WinRT calls, so turn them into .NET tasks and wait.
$asTask = [System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1'
} | Select-Object -First 1
$asTaskWithProgress = [System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperationWithProgress`2'
} | Select-Object -First 1

function Wait-Task($task, [int]$timeoutSeconds) {
    try {
        if (-not $task.Wait($timeoutSeconds * 1000)) { throw "Timed out after $timeoutSeconds seconds." }
    } catch [System.AggregateException] {
        $e = $_.Exception; while ($e.InnerException) { $e = $e.InnerException }
        throw $e
    }
    $task.Result
}
function Await($operation, [type]$resultType, [int]$timeoutSeconds = 120) {
    Wait-Task ($asTask.MakeGenericMethod($resultType).Invoke($null, @($operation))) $timeoutSeconds
}

function Test-Provisioned {
    [bool](Get-AppxProvisionedPackage -Online | Where-Object DisplayName -eq $packageName)
}

function Write-ErrorMessage([int]$code) {
    if ($code -eq $STORE_LICENSE_FAILED) {
        Write-Output "The Microsoft Store couldn't license this app. It may be a paid app."
    } elseif ($code -eq $PACKAGES_IN_USE) {
        Write-Output "The app is open and couldn't be closed. Ask the user to close it, then try again."
    }
}

# Newest installed version for any user, so the script can say whether anything changed.
function Get-InstalledVersion {
    $p = @(Get-AppxPackage -AllUsers -Name $packageName | Sort-Object { [version]$_.Version } -Descending)
    if ($p) { [string]$p[0].Version }
}
$versionBefore = Get-InstalledVersion

$manager = New-Object Windows.ApplicationModel.Store.Preview.InstallControl.AppInstallManager

# Step 1: get a free device license, so no Microsoft account is needed. Best effort, like winget.
try {
    $entitlement = Await ($manager.GetFreeDeviceEntitlementAsync($storeId, '', '')) ([Windows.ApplicationModel.Store.Preview.InstallControl.GetEntitlementResult])
    Write-Output "Device license: $($entitlement.Status)"
} catch {
    Write-Output "Couldn't get a device license: $($_.Exception.Message). Installing anyway."
}

# Step 2: install for all users. If the app is open, the Store closes it, like `winget install --force`.
$options = New-Object Windows.ApplicationModel.Store.Preview.InstallControl.AppInstallOptions
$options.InstallForAllUsers = $true
$options.AllowForcedAppRestart = $true
$options.InstallInProgressToastNotificationMode = 'NoToast'
$options.CompletedInstallToastNotificationMode = 'NoToast'

$itemListType = [System.Collections.Generic.IReadOnlyList[Windows.ApplicationModel.Store.Preview.InstallControl.AppInstallItem]]
try {
    $items = @(Await ($manager.StartProductInstallAsync($storeId, '', 'Fleet', '', $options)) $itemListType)
} catch {
    Write-Output ("The Microsoft Store couldn't start the install: {0} (0x{1:X8})" -f $_.Exception.Message, $_.Exception.HResult)
    Write-ErrorMessage $_.Exception.HResult
    exit $_.Exception.HResult
}

# Step 3: wait for the Store to finish.
$deadline = (Get-Date).AddMinutes($timeoutMinutes)
$resumed = @{}
$errorCode = 0
$last = ''
while ($items.Count -gt 0) {
    $active = 0
    $lines = foreach ($item in $items) {
        $status = $item.GetCurrentStatus()
        $state = [string]$status.InstallState
        if ($state -in @('Error', 'Canceled')) {
            if ($null -ne $status.ErrorCode) { $errorCode = $status.ErrorCode.HResult } else { $errorCode = 1 }
        } elseif ($state -ne 'Completed') {
            $active++
            # Paused installs don't resume on their own when no user is signed in.
            if (($state -like 'Paused*' -or $state -eq 'ReadyToDownload') -and -not $resumed[$item.ProductId]) {
                $manager.Restart($item.ProductId)
                $resumed[$item.ProductId] = $true
            }
        }
        "$($item.PackageFamilyName): $state $([int]$status.PercentComplete)%"
    }
    $line = $lines -join ' | '
    if ($line -ne $last) { Write-Output $line; $last = $line }
    if ($active -eq 0) { break }
    if ((Get-Date) -gt $deadline) {
        Write-Output "Timed out waiting for the Microsoft Store."
        exit 1
    }
    Start-Sleep -Seconds 2
}

if ($errorCode -ne 0) {
    Write-Output ("The Microsoft Store couldn't install the app (0x{0:X8})." -f $errorCode)
    Write-ErrorMessage $errorCode
    exit $errorCode
}

# Step 4: the all-users setup can show up a few seconds after the Store finishes.
for ($i = 0; $i -lt 15 -and -not (Test-Provisioned); $i++) { Start-Sleep -Seconds 2 }

# Step 5: on Windows builds before 26100, the Store can install without setting the app up for
# all users. winget does the same fallback.
if (-not (Test-Provisioned) -and (Get-AppxPackage -AllUsers -Name $packageName)) {
    Write-Output "Setting the app up for all users..."
    try {
        $deploy = $asTaskWithProgress.MakeGenericMethod([Windows.Management.Deployment.DeploymentResult], [Windows.Management.Deployment.DeploymentProgress])
        $null = Wait-Task ($deploy.Invoke($null, @((New-Object Windows.Management.Deployment.PackageManager).ProvisionPackageForAllUsersAsync($packageFamilyName)))) 600
    } catch {
        Write-Output "Couldn't set the app up for all users: $($_.Exception.Message)"
    }
}

# Step 6: verify.
if (-not (Test-Provisioned)) {
    Write-Output "The app isn't installed for all users even though the install reported success."
    exit 1
}
$versionAfter = Get-InstalledVersion
if ($versionBefore -and $versionBefore -eq $versionAfter) {
    Write-Output "Already up to date (version $versionAfter). Installed for all users."
} elseif ($versionBefore) {
    Write-Output "Updated from version $versionBefore to $versionAfter. Installed for all users."
} else {
    Write-Output "Installed version $versionAfter for all users."
}
exit 0
```

The script installs the app for all users on the host, including users who sign in later. If the app is already installed, it installs the latest version, so selecting **Reinstall** in Fleet updates the app. If the app is open during an update, the Microsoft Store closes it, and it doesn't reopen on its own.

## Step 3: Write the uninstall script

Save this as `uninstall-company-portal.ps1`, with the package family name from Step 1.

```powershell
# Uninstalls a Microsoft Store app for all users on the host.

# Change this line for each app (see Step 1).
$packageFamilyName = 'Microsoft.CompanyPortal_8wekyb3d8bbwe'
$packageName = $packageFamilyName.Split('_')[0]

$provisioned = @(Get-AppxProvisionedPackage -Online | Where-Object DisplayName -eq $packageName)
$installed = @(Get-AppxPackage -AllUsers -Name $packageName)
if (-not $provisioned -and -not $installed) {
    Write-Output "App is not installed."
    exit 0
}

# Remove the all-users setup first, so new users don't get the app.
foreach ($p in $provisioned) {
    Remove-AppxProvisionedPackage -Online -PackageName $p.PackageName -AllUsers | Out-Null
}
# Then remove it for every user. If the app is open, its window can stay open until the user closes it.
foreach ($p in $installed) {
    Remove-AppxPackage -Package $p.PackageFullName -AllUsers
}

if (Get-AppxPackage -AllUsers -Name $packageName) {
    Write-Output "The app is still installed for some users."
    exit 1
}
Write-Output "Uninstalled for all users."
exit 0
```

The script removes the app for all users. If the app is open, its window can stay open until the user closes it.

## Step 4: Add the package to Fleet

Add the install script as a script-only package, with the uninstall script from Step 3.

### Fleet UI

1. Go to **Software** and select the fleet.
2. Select **Add software > Custom package** and upload `install-company-portal.ps1`.
3. Expand **Advanced options** and paste the uninstall script into **Uninstall script**.
4. Optionally, turn on **Self-service**.
5. Select **Add software**.

### GitOps

Save both scripts under `lib/windows/scripts/` in your GitOps repo. Then add the package to `fleets/<name>.yml`:

```yaml
software:
  packages:
    - path: ../lib/windows/scripts/install-company-portal.ps1
      display_name: Company Portal
      self_service: true
      uninstall_script:
        path: ../lib/windows/scripts/uninstall-company-portal.ps1
```

> **Note:** Script-only packages don't support automatic install through a policy. Install them from the host's **Host details** page or through self-service.

## Verify

1. Install the app on a test host from **Host details > Software**.
2. Open the install's details. The output ends with "Installed version … for all users."
3. Refetch the host. The app appears in the host's software inventory.
4. Sign in as another user on the host. The app is in their Start menu. On a user's first sign-in after the install, it can take a few seconds to appear.

## Troubleshoot

**The install fails with "The Microsoft Store couldn't license this app."** The app is paid. Only free apps are supported.

**The install fails with "The app is open and couldn't be closed."** Ask the user to close the app, then select **Reinstall**.

**The install fails with "Timed out waiting for the Microsoft Store."** The host couldn't finish the download within 4 minutes. Check the host's network connection, then try again.

**The install fails with "The app isn't installed for all users even though the install reported success."** Check that `$packageFamilyName` in the install script matches the package family name from Step 1.

**`InstallerType` in Step 1 isn't `msstore`.** The app uses its own installer, and the scripts' all-users check fails for it. Adapt the scripts, or add the app as a Fleet-maintained app or custom package.

## Further reading

- [Deploy software packages](https://fleetdm.com/guides/deploy-software-packages)
- [Scripts](https://fleetdm.com/guides/scripts)

<meta name="articleTitle" value="Add Microsoft Store apps to Windows with script-only packages">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-07">
<meta name="description" value="Install free Microsoft Store apps for all users on Windows hosts with Fleet script-only packages, no winget or signed-in user needed.">
