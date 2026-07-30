# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# The Sonos installer is an InstallShield setup that wraps an MSI. The documented
# winget silent switches run the setup silently (/S) and pass /quiet /norestart
# through to the inner MSI (/V) for an unattended, machine-wide install.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S /V/quiet /V/norestart"
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
