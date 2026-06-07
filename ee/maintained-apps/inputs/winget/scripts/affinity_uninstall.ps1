$packageFamilyName = $PACKAGE_ID
$timeoutSeconds = 300  # 5 minute timeout

function Write-Err($prefix, $err) {
  $h = if ($err.Exception) { '0x{0:X8}' -f $err.Exception.HResult } else { 'n/a' }
  Write-Host "$prefix $($err.Exception.Message) [HRESULT: $h]"
}

try {
  $start = Get-Date

  # Best-effort: stop the app so the package isn't locked during removal.
  Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -like "*WindowsApps*Affinity*" } |
    Stop-Process -Force -ErrorAction SilentlyContinue

  # 1) Remove per-user registrations FIRST. Deprovisioning first (as the Slack
  #    template does) can leave the registration mid-teardown, which is the likely
  #    cause of the 1603 here.
  $installed = Get-AppxPackage -AllUsers -PackageTypeFilter Main -ErrorAction SilentlyContinue |
    Where-Object { $_.PackageFamilyName -eq $packageFamilyName }
  foreach ($app in $installed) {
    Write-Host "Removing installed package: $($app.PackageFullName)"
    try {
      Remove-AppxPackage -Package $app.PackageFullName -AllUsers -ErrorAction Stop
    } catch {
      # -AllUsers on a machine-provisioned package is finicky; fall back to current user.
      Write-Err "AllUsers removal failed, retrying without -AllUsers:" $_
      Remove-AppxPackage -Package $app.PackageFullName -ErrorAction Stop
    }
    if ((New-TimeSpan -Start $start).TotalSeconds -gt $timeoutSeconds) { Exit 1603 }
  }

  # 2) Deprovision so it won't reinstall for new users.
  $provisioned = Get-AppxProvisionedPackage -Online -ErrorAction Stop |
    Where-Object { $_.PackageFamilyName -eq $packageFamilyName }
  foreach ($pkg in $provisioned) {
    Write-Host "Removing provisioned package: $($pkg.PackageName)"
    Remove-AppxProvisionedPackage -Online -PackageName $pkg.PackageName -AllUsers -ErrorAction Stop | Out-Null
    if ((New-TimeSpan -Start $start).TotalSeconds -gt $timeoutSeconds) { Exit 1603 }
  }

  Exit 0
} catch {
  Write-Err "Error:" $_
  Exit 1603
}
