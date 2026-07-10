# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Dell's InstallShield (InstallScript) setup installs silently with /S — the
# InstallScript silent switch documented for managed/SYSTEM deployment (SCCM,
# Endpoint Central). The winget manifest's /Silent is an unverified default and
# aborts headless (exit 0x80042000). Run as SYSTEM this installs machine-wide.
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
