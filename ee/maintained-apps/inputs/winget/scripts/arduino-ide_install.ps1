# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# The Arduino IDE MSI is a dual-purpose package (ALLUSERS=2 +
# MSIINSTALLPERUSER=1), so a plain silent install under Fleet's SYSTEM context
# lands in the SYSTEM profile, where some bundled file paths exceed 260
# characters and the install fails with 1603. Force a per-machine install into
# Program Files.

$logFile = "${env:TEMP}/fleet-install-software.log"

# 3010 = reboot required, 1641 = reboot initiated
$successCodes = @(0, 3010, 1641)

try {

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" ALLUSERS=1 MSIINSTALLPERUSER=`"`" /i `"${env:INSTALLER_PATH}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

if ($successCodes -contains $installProcess.ExitCode) {
  Exit 0
}

Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
