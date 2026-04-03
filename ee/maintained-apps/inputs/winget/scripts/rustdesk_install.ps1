# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# RustDesk's --silent-install installs the software then starts the RustDesk
# service/process, which prevents Start-Process -Wait from ever returning. We
# start without -Wait and poll the installer process with a timeout instead.
$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "--silent-install" `
  -PassThru

# Wait up to 3 minutes for the installer process itself to exit.
$timeoutSeconds = 180
$exited = $process.WaitForExit($timeoutSeconds * 1000)

if (-not $exited) {
  Write-Host "Installer process did not exit within ${timeoutSeconds}s, stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 2
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

# Stop the RustDesk background process spawned by the installer so the
# script can return cleanly.
Stop-Process -Name "rustdesk" -Force -ErrorAction SilentlyContinue

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
