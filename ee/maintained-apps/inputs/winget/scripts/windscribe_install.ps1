# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Setup closes a running Windscribe by posting to its window, which it can't
# find from Fleet's SYSTEM context. It then unregisters the previous version
# before failing on the files the app still holds open, leaving Windscribe on
# disk and absent from Programs. Stop the app first; setup stops the service
# and the protocol handlers itself. The VPN connection drops.
$getRunning = { @(Get-Process -Name "Windscribe" -ErrorAction SilentlyContinue) }
$running = & $getRunning
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

# Wait-Process reports a timeout as a non-terminating error, so confirm the app
# really stopped. Setup would unregister the installed copy before failing.
$stillRunning = & $getRunning
if ($stillRunning.Count) {
    Write-Host "Error: Windscribe is still running ($(($stillRunning | ForEach-Object { "$($_.Name) ($($_.Id))" }) -join ', ')). Quit it and try again."
    Exit 1
}

# Windscribe's installer takes exactly these switches: -silent hides its
# interface and -no-auto-start stops it from launching Windscribe when it
# finishes. Any unrecognized switch makes it exit with an error.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "-silent", "-no-auto-start"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
