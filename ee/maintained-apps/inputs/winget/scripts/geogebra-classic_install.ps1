# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# GeoGebra Classic's winget manifest declares Scope: user at the top level, but
# its WiX MSI installer is machine-wide (installs to Program Files (x86)). We run
# the MSI silently; ALLUSERS is set inside the package, so it lands per-machine.

$logFile = "${env:TEMP}/fleet-install-software.log"

try {

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv ${logFile} /i `"${env:INSTALLER_PATH}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

if ($installProcess.ExitCode -eq 3010 -or $installProcess.ExitCode -eq 1641) { Exit 0 }
Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
