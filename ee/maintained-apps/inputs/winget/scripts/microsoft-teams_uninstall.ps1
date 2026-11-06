# Uninstall the new Microsoft Teams (MSIX, PackageFamilyName MSTeams_8wekyb3d8bbwe).
#
# The removal cmdlets can report errors for removals that succeed, and Windows finishes
# removing the package in the background after they return. So the script removes
# best-effort, then waits for the package to be gone and exits 1603 only if it's still
# there when the wait runs out. Already absent counts as success.

$packageFamilyName = $PACKAGE_ID
# Under Fleet's 5-minute script timeout, so a failure exits 1603 instead of being killed.
$timeoutSeconds = 240
$start = Get-Date

# Get-AppxProvisionedPackage objects have no PackageFamilyName. Their PackageName is
# <Name>_<Version>_<Arch>_<ResourceId>_<PublisherId>, and a package name can't contain "_".
function Get-ProvisionedPackage {
  param([string]$pfn)
  $packageNameLike = $pfn -replace '_', '_*_'
  Get-AppxProvisionedPackage -Online -ErrorAction Stop |
    Where-Object { $_.PackageName -like $packageNameLike }
}

# A provisioned package or a signed-in user's copy counts as left behind. A signed-out
# user's copy counts unless Windows marked it "(pending removal)", which finishes at their
# next sign-in. A Staged entry isn't registered for anyone. The queries stop on errors so
# a failed query isn't read as nothing left.
function Test-PackagePresent {
  param([string]$pfn)
  if (@(Get-ProvisionedPackage -pfn $pfn).Count -gt 0) {
    return $true
  }
  $installed = Get-AppxPackage -AllUsers -PackageTypeFilter Main -ErrorAction Stop |
    Where-Object { $_.PackageFamilyName -eq $pfn }
  foreach ($package in @($installed)) {
    foreach ($user in @($package.PackageUserInformation)) {
      $state = $user.InstallState
      if ($state -like 'Staged*') {
        continue
      }
      if ($state -notlike '*(pending removal)' -or
          (Test-Path "Registry::HKEY_USERS\$($user.UserSecurityId.Sid)" -ErrorAction Stop)) {
        return $true
      }
    }
  }
  return $false
}

try {

  # Best-effort: stop the app if it is running so files aren't locked.
  Stop-Process -Name "ms-teams" -Force -ErrorAction SilentlyContinue
  Stop-Process -Name "Teams" -Force -ErrorAction SilentlyContinue

  # Remove the machine-wide provisioning so the package is not re-registered at next sign-in.
  foreach ($pkg in @(Get-ProvisionedPackage -pfn $packageFamilyName)) {
    Write-Host "Removing provisioned package: $($pkg.PackageName)"
    try {
      Remove-AppxProvisionedPackage -Online -PackageName $pkg.PackageName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    } catch {
      Write-Host "Remove-AppxProvisionedPackage reported: $($_.Exception.Message)"
    }
    if ((New-TimeSpan -Start $start).TotalSeconds -gt $timeoutSeconds) { break }
  }

  # Remove the registered package for every user profile.
  $installed = Get-AppxPackage -AllUsers -PackageTypeFilter Main -ErrorAction SilentlyContinue |
    Where-Object { $_.PackageFamilyName -eq $packageFamilyName }
  foreach ($app in @($installed)) {
    Write-Host "Removing installed package: $($app.PackageFullName)"
    try {
      Remove-AppxPackage -Package $app.PackageFullName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    } catch {
      Write-Host "Remove-AppxPackage reported: $($_.Exception.Message)"
    }
    if ((New-TimeSpan -Start $start).TotalSeconds -gt $timeoutSeconds) { break }
  }

  do {
    try {
      if (-not (Test-PackagePresent -pfn $packageFamilyName)) {
        Write-Host "Microsoft Teams ($packageFamilyName) is no longer present."
        Exit 0
      }
    } catch {
      Write-Host "Checking for Microsoft Teams failed, retrying: $($_.Exception.Message)"
    }
    Start-Sleep -Seconds 5
  } while ((New-TimeSpan -Start $start).TotalSeconds -lt $timeoutSeconds)

  Write-Host "Microsoft Teams ($packageFamilyName) is still present after removal attempts."
  Exit 1603

} catch {
  Write-Host "Error: $_"
  Exit 1603
}
