# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Zen Browser ships a Mozilla/NSIS-based machine-scope installer.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Add arguments to install silently machine-wide.
# /S          -> silent
# /PreventRebootRequired=true -> suppress reboot prompts
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S /PreventRebootRequired=true"
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
