# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Comet ships a Chromium/Omaha-based machine-scope installer
# (comet_*_system.exe). Fleet runs as SYSTEM, so it installs machine-wide.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Add arguments to install silently machine-wide.
# --install --silent -> silent machine-scope install
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--install --silent"
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
