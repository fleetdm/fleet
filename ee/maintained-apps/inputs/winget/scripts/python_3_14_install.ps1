# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# python.org's Windows installer is a WiX "Burn" bundle. "/quiet /norestart"
# runs it silently. Fleet installs run as SYSTEM, so force a per-machine install
# (InstallAllUsers=1) instead of the default per-user one, and add Python to PATH.

$exeFilePath = "${env:INSTALLER_PATH}"
# 0 = success, 3010 = success (reboot required), 1641 = success (reboot initiated).
$ExpectedExitCodes = @(0, 3010, 1641)

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/quiet /norestart InstallAllUsers=1 PrependPath=1"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
