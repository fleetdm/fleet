# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Firefox's full installer is NSIS-based; /S installs silently and machine-wide.
# /RegisterDefaultAgent=false skips registering Mozilla's Default Browser Agent
# scheduled task, which can't be registered from the SYSTEM context this script
# runs in: the attempt fails with 0x80070534 ("No mapping between account names
# and security IDs") in the Application event log and can surface a firefox.exe
# error dialog on the user's first launch.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S /RegisterDefaultAgent=false"
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
