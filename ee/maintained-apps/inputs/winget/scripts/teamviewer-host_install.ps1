# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# TeamViewer Host ships as an NSIS installer, which takes "/S" for a silent
# install. The MSI underneath sets ALLUSERS=1, so this always installs
# machine-wide.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S"
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
