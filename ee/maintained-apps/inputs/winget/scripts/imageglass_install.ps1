# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# The ImageGlass MSI is dual-scope (ALLUSERS=2). Pass ALLUSERS=1 to force a
# per-machine install under Fleet's SYSTEM context.

$logFile = "${env:TEMP}/fleet-install-software.log"

try {

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv ${logFile} ALLUSERS=1 /i `"${env:INSTALLER_PATH}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

if ($installProcess.ExitCode -eq 3010 -or $installProcess.ExitCode -eq 1641) { Exit 0 }
Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
