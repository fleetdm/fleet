# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# The Delinea Connection Manager MSI is a dual-purpose package: its Property
# table sets ALLUSERS=2 + MSIINSTALLPERUSER=1, so a plain silent install under
# Fleet's SYSTEM context lands per-user (in the SYSTEM profile) despite the
# winget manifest's machine scope. Force a true per-machine install by setting
# ALLUSERS=1 and clearing MSIINSTALLPERUSER.

$logFile = "${env:TEMP}/fleet-install-software.log"

try {

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv ${logFile} ALLUSERS=1 MSIINSTALLPERUSER=`"`" /i `"${env:INSTALLER_PATH}`"" `
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
