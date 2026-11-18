# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# With GIMP open, Setup leaves its locked files to be replaced at the next
# reboot, and later installs exit with code 8 until then. Fleet runs as SYSTEM,
# outside the user's session, so it can't ask the app to close: stop it first.
# Unsaved changes are lost.
$running = Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -like "*\GIMP 3\*" }
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

# Inno Setup installer with /ALLUSERS for machine-scope installation
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /ALLUSERS"
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
