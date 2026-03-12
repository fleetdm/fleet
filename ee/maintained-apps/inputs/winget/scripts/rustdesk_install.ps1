# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

# RustDesk's silent installer spawns a persistent service process, so we use
# WaitForExit with a timeout instead of -Wait to avoid blocking indefinitely.
$timeoutSeconds = 120

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "--silent-install" `
  -PassThru

$exited = $process.WaitForExit($timeoutSeconds * 1000)
if (-not $exited) {
  Write-Host "Installer did not exit within $timeoutSeconds seconds; killing process."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 2
}

# Kill any lingering RustDesk service processes spawned by the installer
Stop-Process -Name "rustdesk" -Force -ErrorAction SilentlyContinue

Write-Host "Install complete."
Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
}
