# Learn more about .msi install scripts:
# http://fleetdm.com/learn-more-about/msi-install-scripts
#
# Egnyte's MSI has a LaunchCondition that fails (1603) when reboot
# suppression is requested (/norestart => REBOOT=ReallySuppress) unless
# ED_UPDATE_ON_BOOT=1 is also passed, which schedules the CBFS driver
# update at next boot instead of forcing an immediate reboot.

$logFile = "${env:TEMP}/fleet-install-software.log"
$msiFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "msiexec.exe"
  ArgumentList = "/i `"$msiFilePath`" /quiet /norestart ED_UPDATE_ON_BOOT=1 /lv `"$logFile`""
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"

# MSI reboot-required success codes.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

if ($exitCode -ne 0) {
  Get-Content $logFile -Tail 500
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
