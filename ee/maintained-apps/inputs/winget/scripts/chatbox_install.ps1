# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Chatbox uses an electron-builder NSIS installer. It defaults to a per-user
# install, so /allusers is required for a machine-wide install alongside the
# NSIS /S silent flag.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/allusers /S"
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
