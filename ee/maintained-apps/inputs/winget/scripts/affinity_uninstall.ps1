$packageFamilyName = $PACKAGE_ID
$timeoutSeconds = 300

try {

  $start = Get-Date

  $provisioned = Get-AppxProvisionedPackage -Online -ErrorAction Stop |
    Where-Object { $_.PackageFamilyName -eq $packageFamilyName }
  foreach ($pkg in $provisioned) {
    Write-Host "Removing provisioned package: $($pkg.PackageName)"
    Remove-AppxProvisionedPackage -Online -PackageName $pkg.PackageName -AllUsers -ErrorAction Stop
    $elapsed = (New-TimeSpan -Start $start).TotalSeconds
    if ($elapsed -gt $timeoutSeconds) {
      Exit 1603
    }
  }

  $installed = Get-AppxPackage -AllUsers -PackageTypeFilter Main -ErrorAction SilentlyContinue |
    Where-Object { $_.PackageFamilyName -eq $packageFamilyName }
  foreach ($app in $installed) {
    Write-Host "Removing installed package: $($app.PackageFullName)"
    Remove-AppxPackage -Package $app.PackageFullName -AllUsers -ErrorAction Stop
    $elapsed = (New-TimeSpan -Start $start).TotalSeconds
    if ($elapsed -gt $timeoutSeconds) {
      Exit 1603
    }
  }

  Exit 0

} catch {
  $h = if ($_.Exception) { '0x{0:X8}' -f $_.Exception.HResult } else { 'n/a' }
  Write-Host "Error: $($_.Exception.Message) [HRESULT: $h]"
  Exit 1603
}
