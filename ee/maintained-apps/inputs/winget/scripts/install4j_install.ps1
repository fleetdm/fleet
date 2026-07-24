# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# install4j ships as an install4j-built installer (bundled JRE). -q runs it
# unattended; suppressUnattendedReboot avoids an automatic reboot.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "-q -Dinstall4j.suppressUnattendedReboot=true" -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
