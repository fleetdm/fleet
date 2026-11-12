# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Setup fails with exit code 5 when WinSCP's files are in use. Fleet runs as
# SYSTEM, outside the user's session, so it can't ask the app to close: stop it
# first. Unsaved changes are lost.
$running = Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -like "*\WinSCP\*" }
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

# WinSCP uses an Inno Setup installer. /ALLUSERS forces a machine-wide install
# (the default is per-user, which would land in the SYSTEM profile when Fleet
# runs as SYSTEM). /VERYSILENT /SUPPRESSMSGBOXES /NORESTART are the standard
# Inno Setup silent switches.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/ALLUSERS /VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# A file can stay locked for a moment after the app is stopped. Setup rolls
# back cleanly on exit code 5, so retry once.
if ($exitCode -eq 5 -and $running) {
    Write-Host "Install exit code: 5. Retrying in 10 seconds."
    Start-Sleep -Seconds 10
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
}

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
