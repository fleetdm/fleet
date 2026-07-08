# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Add arguments to install silently (CloudShow uses an Inno Setup-based
# installer; /LAUNCHAFTER=0 keeps it from launching post-install)
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /LAUNCHAFTER=0"
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
