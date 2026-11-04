# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Citrix Workspace app ships a vendor bootstrapper (CitrixWorkspaceApp.exe).
# /silent installs machine-wide without UI; /noreboot keeps it from rebooting the
# host; /AutoUpdateCheck=disabled leaves updates to Fleet rather than the app.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/silent", "/noreboot", "/AutoUpdateCheck=disabled"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { $exitCode = 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
