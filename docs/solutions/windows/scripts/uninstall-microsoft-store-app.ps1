# Uninstalls a Microsoft Store app for all users on the host.

# Change this line for each app. Fleet's guide shows how to find the package family name:
# https://fleetdm.com/guides/build-your-own-windows-self-service-with-winget-and-script-only-packages-guide
$packageFamilyName = 'Microsoft.CompanyPortal_8wekyb3d8bbwe'
$packageName = $packageFamilyName.Split('_')[0]

# True if the app is still set up for new users or still registered for a user.
# A Staged entry isn't registered for anyone, and a "pending removal" entry for a signed-out
# user is removed at their next sign-in, so neither counts. The queries stop on errors, so a
# failed query isn't read as "not installed".
function Test-AppRemains {
    if (@(Get-AppxProvisionedPackage -Online -ErrorAction Stop | Where-Object DisplayName -eq $packageName).Count -gt 0) {
        return $true
    }
    foreach ($package in @(Get-AppxPackage -AllUsers -Name $packageName -ErrorAction Stop)) {
        foreach ($user in @($package.PackageUserInformation)) {
            $state = [string]$user.InstallState
            if ($state -like 'Staged*') { continue }
            if ($state -notlike '*(pending removal)' -or
                (Test-Path "Registry::HKEY_USERS\$($user.UserSecurityId.Sid)")) {
                return $true
            }
        }
    }
    return $false
}

try {
    $provisioned = @(Get-AppxProvisionedPackage -Online -ErrorAction Stop | Where-Object DisplayName -eq $packageName)
    $installed = @(Get-AppxPackage -AllUsers -Name $packageName -ErrorAction Stop)
} catch {
    Write-Output "Couldn't check whether the app is installed: $($_.Exception.Message)"
    exit 1
}
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

try {
    $remains = Test-AppRemains
} catch {
    Write-Output "Couldn't verify the uninstall: $($_.Exception.Message)"
    exit 1
}
if ($remains) {
    Write-Output "The app is still installed for some users."
    exit 1
}
Write-Output "Uninstalled for all users."
exit 0
