# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Garmin Express is a WiX Burn bundle; /silent installs it unattended machine-wide.
# The app self-updates after install, so Fleet's version is a nominal baseline.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/silent" -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

# 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
