# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Opera's installer uses /silent for silent installation. /allusers=1 forces a
# machine-wide (all users) install so it is visible to Fleet's SYSTEM context.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/silent /allusers=1"
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
