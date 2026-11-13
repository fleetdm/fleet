# Uninstalls a Microsoft Store app for all users on the host.

# Change this line for each app. Fleet's guide shows how to find the package family name:
# https://fleetdm.com/guides/build-your-own-windows-self-service-with-winget-and-script-only-packages-guide
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
