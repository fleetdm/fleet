# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Mullvad VPN uses an NSIS installer that always installs for all users.
# /S runs it silently. It force-quits a running Mullvad VPN and restarts
# the Mullvad VPN service.

$exeFilePath = "${env:INSTALLER_PATH}"
$timeoutSeconds = 600

try {

# This x64 installer shows a message box on Windows on Arm, even when silent.
$nativeArch = (Get-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment').PROCESSOR_ARCHITECTURE
if ($nativeArch -eq 'ARM64') {
  Write-Host "Error: This is the x64 build of Mullvad VPN. It can't be installed on Windows on Arm."
  Exit 1
}

$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/S" -PassThru
# Keeps .ExitCode readable after the process ends.
$null = $process.Handle

# Some installer failures show a message box even when silent.
if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
  Write-Host "Installer did not exit within $timeoutSeconds seconds; terminating it."
  & taskkill.exe /PID $process.Id /T /F 2>&1 | Write-Host
  Exit 1603
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
