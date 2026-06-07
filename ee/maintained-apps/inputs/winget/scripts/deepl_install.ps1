# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# DeepL Setup (Inno/Zero Install based):
#   --verysilent : silent install, no UI
#   --no-run     : do not launch the app after install
#   --machine    : install machine-wide (per the winget machine-scope installer)
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "--verysilent --no-run --machine"
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
