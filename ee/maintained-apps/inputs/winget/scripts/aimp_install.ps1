# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# AIMP uses its own setup; silent install switches per the winget manifest
# are /AUTO /SILENT (not the Inno-style /VERYSILENT).
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/AUTO /SILENT"
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
