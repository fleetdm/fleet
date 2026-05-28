# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Tableau Desktop ships as a WiX Burn bundle.
#   /quiet /norestart   - standard Burn silent install
#   ACCEPTEULA=1        - required by Tableau (winget Custom switch)
#   SKIPAPPLICATIONLAUNCH=1 - don't launch after install

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/quiet /norestart ACCEPTEULA=1 SKIPAPPLICATIONLAUNCH=1"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"

# msiexec reboot-required success codes (Burn bundles propagate these).
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Exit 0
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
