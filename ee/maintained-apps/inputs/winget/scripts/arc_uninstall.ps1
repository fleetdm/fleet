$timeoutSeconds = 300  # 5 minute timeout

function ShouldRemoveArcPackage {
  param([Parameter(Mandatory=$true)]$pkg)
  try {
    $name = [string]$pkg.Name
    $family = [string]$pkg.PackageFamilyName
    $publisher = [string]$pkg.Publisher

    if ($family -and ($family -like "TheBrowserCompany.Arc_*")) { return $true }
    if ($name -and ($name -like "TheBrowserCompany.Arc")) { return $true }
    if ($publisher -and ($publisher -like "*Browser Company*")) { return $true }
  } catch {}
  return $false
}

try {

  $start = Get-Date

  # Best-effort: close app if running (name may vary)
  Stop-Process -Name "Arc" -Force -ErrorAction SilentlyContinue

  $provisioned = Get-AppxProvisionedPackage -Online -ErrorAction Stop | Where-Object {
    ($_.PackageFamilyName -and ($_.PackageFamilyName -like "TheBrowserCompany.Arc_*")) -or
    ($_.DisplayName -and ($_.DisplayName -like "TheBrowserCompany.Arc")) -or
    ($_.PackageName -and ($_.PackageName -like "TheBrowserCompany.Arc*"))
  }

  foreach ($pkg in $provisioned) {
    Write-Host "Removing provisioned package: $($pkg.PackageName)"
    Remove-AppxProvisionedPackage -Online -PackageName $pkg.PackageName -AllUsers -ErrorAction Stop | Out-String | Write-Host
    $elapsed = (New-TimeSpan -Start $start).TotalSeconds
    if ($elapsed -gt $timeoutSeconds) { Exit 1603 }
  }

  $installed = Get-AppxPackage -AllUsers -PackageTypeFilter Main -ErrorAction SilentlyContinue | Where-Object {
    ShouldRemoveArcPackage $_
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
