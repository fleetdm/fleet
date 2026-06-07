# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# OneDriveSetup.exe installs silently with /silent. /allusers forces a
# machine-wide install so OneDrive is visible to Fleet's SYSTEM context
# (the default behavior is per-user).
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/allusers /silent"
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# Prints the exit code
Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
