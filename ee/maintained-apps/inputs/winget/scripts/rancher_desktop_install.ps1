# Learn more about .msi install scripts:
# http://fleetdm.com/learn-more-about/msi-install-scripts
#
# Rancher Desktop's winget manifest documents a Custom switch of
# WSLINSTALLED=1 — this property tells the installer to skip the
# WSL setup step (which requires a reboot). Pass it via msiexec.

$msiFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "msiexec.exe"
  ArgumentList = "/i `"$msiFilePath`" /quiet /norestart WSLINSTALLED=1"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
if ($exitCode -eq 3010) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
