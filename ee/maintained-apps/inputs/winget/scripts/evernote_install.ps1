# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Evernote uses an NSIS-based installer; /allusers installs machine-wide, /S for silent
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/allusers /S"
  PassThru = $true
  Wait = $true
  NoNewWindow = $true
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
