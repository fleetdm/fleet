# GoToMeeting MSI machine-wide install.
#   G2MINSTALLFORALLUSERS=1 installs for all users (the installer defaults to
#   per-user, G2MINSTALLFORALLUSERS=0). Fleet runs elevated, so this produces a
#   machine-wide install.

$logFile = "${env:TEMP}/fleet-install-software.log"

try {

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv ${logFile} /i `"${env:INSTALLER_PATH}`" G2MINSTALLFORALLUSERS=1" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
