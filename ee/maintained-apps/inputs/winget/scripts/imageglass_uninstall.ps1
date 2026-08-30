$timeoutSeconds = 300  # 5 minute timeout

# Match on the package identity (DuongDieuPhap.ImageGlass) rather than a bare "ImageGlass"
# wildcard, so the separate DuongDieuPhap.ImageGlass.ProBusiness package is left alone.
function ShouldRemoveImageGlassPackage {
  param([Parameter(Mandatory=$true)]$pkg)
  try {
    $name = [string]$pkg.Name
    $family = [string]$pkg.PackageFamilyName

    if ($family -and ($family -like "DuongDieuPhap.ImageGlass_*")) { return $true }
    if ($name -and ($name -like "DuongDieuPhap.ImageGlass")) { return $true }
  } catch {}
  return $false
}

try {

  $start = Get-Date

  # Best-effort: close app if running (name may vary)
  Stop-Process -Name "ImageGlass" -Force -ErrorAction SilentlyContinue

  $provisioned = Get-AppxProvisionedPackage -Online -ErrorAction Stop | Where-Object {
    ($_.PackageFamilyName -and ($_.PackageFamilyName -like "DuongDieuPhap.ImageGlass_*")) -or
    ($_.DisplayName -and ($_.DisplayName -like "DuongDieuPhap.ImageGlass")) -or
    ($_.PackageName -and ($_.PackageName -like "DuongDieuPhap.ImageGlass_*"))
  }

  foreach ($pkg in $provisioned) {
    Write-Host "Removing provisioned package: $($pkg.PackageName)"
    Remove-AppxProvisionedPackage -Online -PackageName $pkg.PackageName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    $elapsed = (New-TimeSpan -Start $start).TotalSeconds
    if ($elapsed -gt $timeoutSeconds) { Exit 1603 }
  }

  $installed = Get-AppxPackage -AllUsers -PackageTypeFilter Main -ErrorAction SilentlyContinue | Where-Object {
    ShouldRemoveImageGlassPackage $_
  }

  foreach ($app in $installed) {
    Write-Host "Removing installed package: $($app.PackageFullName)"
    Remove-AppxPackage -Package $app.PackageFullName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    $elapsed = (New-TimeSpan -Start $start).TotalSeconds
    if ($elapsed -gt $timeoutSeconds) { Exit 1603 }
  }

  Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1603
}
