# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Opera ships a 7-Zip self-extracting installer. /silent installs without UI and
# /allusers=1 forces a per-machine install (Program Files + HKLM) so the app is
# reachable from Fleet's SYSTEM context. /launchopera=0 keeps the installer from
# opening the browser when it finishes.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/silent", "/allusers=1", "/launchopera=0", "/setdefaultbrowser=0"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
