# Installs a Microsoft Store app for all users on the host.
# If the app is already installed, the Microsoft Store installs the latest version.

# Change these two lines for each app. The Store ID is the last part of the app's
# Microsoft Store URL. Fleet's guide shows how to find the package family name:
# https://fleetdm.com/guides/build-your-own-windows-self-service-with-winget-and-script-only-packages-guide
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
