# Uninstall the new Microsoft Teams (MSIX, PackageFamilyName MSTeams_8wekyb3d8bbwe).
#
# The Fleet agent runs as Local System. Removing the system-provisioned new Teams the
# naive way (Remove-AppxProvisionedPackage / Remove-AppxPackage with -ErrorAction Stop)
# fails with exit 1603 / "Removal failed. Please contact your software vendor." because a
# single non-fatal cmdlet error aborts the whole script even when the package does end up
# removed. Instead we remove best-effort (no -ErrorAction Stop abort), then re-check with
# Get-AppxPackage and exit 0 when the package is actually gone. Treat "already absent" as
# success so the script is idempotent.

$packageFamilyName = $PACKAGE_ID
$timeoutSeconds = 300  # 5 minute timeout
$start = Get-Date

function Test-PackagePresent {
  param([string]$pfn)
  $prov = Get-AppxProvisionedPackage -Online -ErrorAction SilentlyContinue |
    Where-Object { $_.PackageFamilyName -eq $pfn }
  $inst = Get-AppxPackage -AllUsers -PackageTypeFilter Main -ErrorAction SilentlyContinue |
    Where-Object { $_.PackageFamilyName -eq $pfn }
  return (($prov | Measure-Object).Count -gt 0) -or (($inst | Measure-Object).Count -gt 0)
}

try {

  # Best-effort: stop the app if it is running so files aren't locked.
  Stop-Process -Name "ms-teams" -Force -ErrorAction SilentlyContinue
  Stop-Process -Name "Teams" -Force -ErrorAction SilentlyContinue

  # Remove the machine-wide provisioning so the package is not re-registered at next sign-in.
  $provisioned = Get-AppxProvisionedPackage -Online -ErrorAction SilentlyContinue |
    Where-Object { $_.PackageFamilyName -eq $packageFamilyName }
  foreach ($pkg in $provisioned) {
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
  foreach ($app in $installed) {
    Write-Host "Removing installed package: $($app.PackageFullName)"
    try {
      Remove-AppxPackage -Package $app.PackageFullName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    } catch {
      Write-Host "Remove-AppxPackage reported: $($_.Exception.Message)"
    }
    if ((New-TimeSpan -Start $start).TotalSeconds -gt $timeoutSeconds) { break }
  }

  # Verify the outcome rather than trusting cmdlet exit status: a non-fatal error above is
  # fine as long as the package is actually gone.
  if (-not (Test-PackagePresent -pfn $packageFamilyName)) {
    Write-Host "Microsoft Teams ($packageFamilyName) is no longer present."
    Exit 0
  }

  Write-Host "Microsoft Teams ($packageFamilyName) is still present after removal attempts."
  Exit 1603

} catch {
  Write-Host "Error: $_"
  if (-not (Test-PackagePresent -pfn $packageFamilyName)) {
    Write-Host "Package is absent despite the error; treating as success."
    Exit 0
  }
  Exit 1603
}
