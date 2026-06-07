# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# ExpressVPN ships a WiX burn bundle. Silent switches verified against the
# winget InstallerSwitches (Silent: /install /quiet /norestart) and
# silentinstallhq.com (same command). The burn engine relaunches itself
# elevated; Start-Process -Wait waits on the bootstrapper, which exits once the
# chained packages finish.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/install /quiet /norestart"
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"

# WiX burn returns 3010 when a reboot is required; treat reboot codes as success.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Write-Host "Install requires a reboot (exit code $exitCode); treating as success."
  Exit 0
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
