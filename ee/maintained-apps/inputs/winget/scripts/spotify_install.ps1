# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Add arguments to install silently
# Spotify uses /silent /skip-app-launch for silent installation
# Note: elevationProhibited - do not use -Verb RunAs
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/silent", "/skip-app-launch"
  PassThru = $true
  Wait = $true
}
    
# Start process and track exit code
# Exit code 26 indicates packageInUse, which is expected in some scenarios
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# Prints the exit code
Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}

