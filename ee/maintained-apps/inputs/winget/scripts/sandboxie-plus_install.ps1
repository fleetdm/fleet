# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Sandboxie-Plus uses an Inno Setup installer. Installing over an existing
# version closes the Sandboxie-Plus window, ends every sandboxed program, and
# reinstalls the Sandboxie driver and service. The installer's optional
# download of newer templates and translations is turned off.

$exeFilePath = "${env:INSTALLER_PATH}"
$logFile = Join-Path ([System.IO.Path]::GetTempPath()) "fleet-sandboxie-plus-install.log"
$timeoutSeconds = 480

try {

# The installer asks a question it can't skip in these two cases, even when
# silent, so stop before starting it.
$osVersion = [Environment]::OSVersion.Version
if ($osVersion.Major -eq 10 -and $osVersion.Build -ge 10240 -and $osVersion.Build -le 14393) {
  Write-Host "Error: Sandboxie-Plus doesn't support Windows 10 versions 1507-1607 or Windows Server 2016. This host is on build $($osVersion.Build)."
  Exit 1
}

$classic = Get-ItemProperty -LiteralPath 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Sandboxie' -ErrorAction SilentlyContinue
if ($classic -and $classic.UninstallString) {
  Write-Host "Error: Sandboxie Classic is installed. Uninstall it, then install Sandboxie-Plus again."
  Exit 1
}

$installArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /ALLUSERS /MERGETASKS=!RefreshBuild,!DesktopIcon /LOG=`"$logFile`""
$process = Start-Process -FilePath "$exeFilePath" -ArgumentList $installArgs -PassThru
# Keeps .ExitCode readable after the process ends.
$null = $process.Handle

# A driver or service error shows a message box even when silent.
if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
  Write-Host "Installer did not exit within $timeoutSeconds seconds; terminating it."
  & taskkill.exe /PID $process.Id /T /F 2>&1 | Write-Host
  Get-Content -LiteralPath $logFile -Tail 40 -ErrorAction SilentlyContinue
  Exit 1603
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

# 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
if ($exitCode -ne 0 -and $exitCode -ne 3010 -and $exitCode -ne 1641) {
  Get-Content -LiteralPath $logFile -Tail 40 -ErrorAction SilentlyContinue
  Exit $exitCode
}

$service = Get-Service -Name 'SbieSvc' -ErrorAction SilentlyContinue
if (-not $service -or $service.Status -ne 'Running') {
  Write-Host "Warning: the Sandboxie service (SbieSvc) is not running."
}

Remove-Item -LiteralPath $logFile -Force -ErrorAction SilentlyContinue
Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
}
