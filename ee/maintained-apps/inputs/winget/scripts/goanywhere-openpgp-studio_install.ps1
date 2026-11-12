# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# GoAnywhere OpenPGP Studio ships as an install4j installer (bundled JRE). -q
# runs it unattended; suppressUnattendedReboot avoids an automatic reboot.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Setup exits with code 1 while OpenPGP Studio is running. Fleet runs as SYSTEM,
# outside the user's session, so it can't ask the app to close: stop it
# first. Unsaved changes are lost.
$running = Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -like "*\GoAnywhere OpenPGP Studio\*" }
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "-q -Dinstall4j.suppressUnattendedReboot=true" -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
