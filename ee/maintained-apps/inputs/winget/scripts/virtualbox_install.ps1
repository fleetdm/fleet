# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Oracle VirtualBox ships an exe wrapper around a WiX MSI. The winget manifest
# documents the silent install as: --silent with -msiparams REBOOT=ReallySuppress.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--silent -msiparams REBOOT=ReallySuppress"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
# Treat 3010 (reboot required) as success.
if ($exitCode -eq 3010) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
