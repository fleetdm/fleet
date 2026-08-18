# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# GeoGebra Classic ships both a user-scoped EXE and a machine-wide WiX MSI. We
# use the MSI, which installs to Program Files (x86). We run it silently;
# ALLUSERS is set inside the package, so it lands per-machine.

$logFile = "${env:TEMP}/fleet-install-software.log"

try {

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" /i `"${env:INSTALLER_PATH}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

if ($installProcess.ExitCode -eq 3010 -or $installProcess.ExitCode -eq 1641) { Exit 0 }
Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
