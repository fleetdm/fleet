# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# The Azure Functions Core Tools MSI is machine-scope per its winget manifest,
# but its Property table has no ALLUSERS row. When msiexec runs in Fleet's
# SYSTEM context that would default to a per-user install into the SYSTEM
# profile, so we pass ALLUSERS=1 explicitly to force a per-machine install.

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
