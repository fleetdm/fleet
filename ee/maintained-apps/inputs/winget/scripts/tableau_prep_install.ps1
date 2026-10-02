# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Tableau Prep Builder ships as a WiX Burn bundle.
#   ACCEPTEULA=1            - required; the install fails without it
#   SKIPAPPLICATIONLAUNCH=1 - don't open Prep Builder after installing

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

# 3010 and 1641 mean the install succeeded and a reboot is required to finish.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Exit 0
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
