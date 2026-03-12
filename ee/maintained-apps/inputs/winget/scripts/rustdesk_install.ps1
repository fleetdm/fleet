# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

# RustDesk's silent installer returns exit code 1 even on success
$ExpectedExitCodes = @(0, 1)

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--silent-install"
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# Prints the exit code
Write-Host "Install exit code: $exitCode"

# Treat acceptable exit codes as success
if ($ExpectedExitCodes -contains $exitCode) {
  Exit 0
} else {
  Exit $exitCode
}

} catch {
  Write-Host "Error: $_"
  Exit 1
}
