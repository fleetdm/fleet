$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 1641, 3010, 1223)
$timeoutSeconds = 180

# Inno Setup writes its uninstall registry key (DockManager_is1) before running
# its [Run] section. That post-install step launches the Dock Manager app/service,
# which keeps the installer process alive indefinitely under the SYSTEM context.
# So we wait for the install to register itself, then stop any lingering installer
# or app processes instead of blocking on them.
$regPaths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\DockManager_is1',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\DockManager_is1'
)

function Test-Installed {
  foreach ($p in $regPaths) { if (Test-Path $p) { return $true } }
  return $false
}

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /NOCANCEL /SP-"
  PassThru = $true
}

$process = Start-Process @processOptions

$elapsed = 0
while ($elapsed -lt $timeoutSeconds) {
  if ($process.HasExited) { break }
  if (Test-Installed) { break }
  Start-Sleep -Seconds 3
  $elapsed += 3
}

# If the installer is still running (blocked on its post-install [Run] step),
# stop the launched app and then the installer so the script can return.
if (-not $process.HasExited) {
  Stop-Process -Name "dockmgr" -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 3
  if (-not $process.HasExited) {
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  }
}

if (Test-Installed) {
  Write-Host "Dock Manager installed successfully."
  Exit 0
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"
if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
