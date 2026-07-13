# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# CrisisGo is an InstallShield Basic MSI whose InstallExecuteSequence contains
# a "must run setup.exe" guard conditioned on NOT ISSETUPDRIVEN. The vendor
# ships this bare MSI for network deployment and winget's sandbox installs it
# silently, but we pass ISSETUPDRIVEN=1 explicitly so the guard can never fire.

$logFile = "${env:TEMP}/fleet-install-software.log"

try {

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv ${logFile} ISSETUPDRIVEN=1 /i `"${env:INSTALLER_PATH}`"" `
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
