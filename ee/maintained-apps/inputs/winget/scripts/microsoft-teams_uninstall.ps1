$packageFamilyName = $PACKAGE_ID
$timeoutSeconds = 300  # 5 minute timeout

try {

  $start = Get-Date
  $provisioned = Get-AppxProvisionedPackage -Online -ErrorAction Stop |
    Where-Object { $_.PackageFamilyName -eq $packageFamilyName }
  foreach ($pkg in $provisioned) {
    Remove-AppxProvisionedPackage -Online -PackageName $pkg.PackageName -AllUsers -ErrorAction Stop
    $elapsed = (New-TimeSpan -Start $start).TotalSeconds
    if ($elapsed -gt $timeoutSeconds) {
      Exit 1603
    }
  }

  $installed = Get-AppxPackage -AllUsers -PackageTypeFilter Main -ErrorAction SilentlyContinue |
    Where-Object { $_.PackageFamilyName -eq $packageFamilyName }
  foreach ($app in $installed) {
    Remove-AppxPackage -Package $app.PackageFullName -AllUsers -ErrorAction Stop
    $elapsed = (New-TimeSpan -Start $start).TotalSeconds
    if ($elapsed -gt $timeoutSeconds) {
      Exit 1603
    }
  }

  Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1603
}
