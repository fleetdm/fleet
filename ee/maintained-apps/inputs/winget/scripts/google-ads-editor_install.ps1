# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# The Google Ads Editor MSI ships as both a per-user and a per-machine package
# (same binary; the machine variant sets ALLUSERS=1). A plain silent install
# under Fleet's SYSTEM context can land per-user (in the SYSTEM profile), so we
# force a true per-machine install by passing ALLUSERS=1 explicitly.

$logFile = "${env:TEMP}/fleet-install-software.log"

try {

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv ${logFile} ALLUSERS=1 /i `"${env:INSTALLER_PATH}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

# 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
if ($installProcess.ExitCode -eq 3010 -or $installProcess.ExitCode -eq 1641) {
  Exit 0
}
Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
