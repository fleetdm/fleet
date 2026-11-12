# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# MindManager redistributable bootstrapper wraps an MSI. The documented silent
# switches pass /qn through to the inner MSI for an unattended, machine-wide install.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S /v/qn"
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
