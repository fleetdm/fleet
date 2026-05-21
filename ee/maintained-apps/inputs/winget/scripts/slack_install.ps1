# MSIX: provision machine-wide, then register for the current user when possible so inventory
# (osquery programs) and FMA validation are more likely to see the app immediately.

try {

  $msixPath = $env:INSTALLER_PATH
  if (-not $msixPath) {
    throw "INSTALLER_PATH is not set"
  }

  Write-Host "Provisioning MSIX for all users..."
  $result = Add-AppxProvisionedPackage -Online -PackagePath $msixPath -SkipLicense -ErrorAction Stop
  $result | Out-String | Write-Host

  # Per-user registration helps ARP/programs visibility on hosts where validation runs with a user session.
  try {
    Write-Host "Registering MSIX for current user (best-effort)..."
    Add-AppxPackage -Path $msixPath -ErrorAction Stop | Out-String | Write-Host
  } catch {
    Write-Host "Add-AppxPackage skipped or failed (provisioned install may still be valid): $($_.Exception.Message)"
  }

  Start-Sleep -Seconds 5
  Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
}
