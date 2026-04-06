try {

  $result = Add-AppxProvisionedPackage -Online -PackagePath $env:INSTALLER_PATH -SkipLicense -ErrorAction Stop
  $result | Out-String | Write-Host
  Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
}
