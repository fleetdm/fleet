# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"
$installDir = "C:\Program Files\Mozilla Firefox"

try {

# Firefox's full installer is NSIS-based; /S installs silently and machine-wide.
# /RegisterDefaultAgent=false: the Default Browser Agent task can't be registered
# from SYSTEM context (0x80070534 error + firefox.exe dialog on first launch).
$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/S /RegisterDefaultAgent=false" -PassThru

# Wait for the installer to exit so operations that run right after (uninstall,
# upgrade) don't race it. Its post-install background tasks can keep it alive,
# so cap the wait and accept a landed firefox.exe as success.
if ($process.WaitForExit(300000)) {
  $exitCode = $process.ExitCode
  Write-Host "Install exit code: $exitCode"
  Exit $exitCode
}

if (Test-Path "$installDir\firefox.exe") {
  Write-Host "Installer still running after 300s; Firefox ESR is installed"
  Exit 0
}

Write-Host "Timed out waiting for Firefox ESR to install"
Exit 1

} catch {
  Write-Host "Error: $_"
  Exit 1
}
