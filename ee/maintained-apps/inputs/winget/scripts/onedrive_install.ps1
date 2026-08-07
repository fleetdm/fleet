# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# OneDriveSetup.exe performs a per-machine install with "/allusers /silent"
# (switches verified against the winget InstallerSwitches Custom: /allusers and
# silentinstallhq.com). The catch: OneDriveSetup.exe spawns several child
# processes and starts the resident OneDrive.exe, so a plain Start-Process -Wait
# can wait indefinitely and hit the CI step timeout. Instead, start the
# installer, then poll for the per-machine install to land (registry uninstall
# key + the all-users binary) and return success as soon as it appears.

$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/allusers /silent" -PassThru

# Per-machine OneDrive registers an uninstall key and drops OneDrive.exe under
# Program Files (x86) (or Program Files on x86 OS).
$uninstallKey = "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\OneDriveSetup.exe"
$exePaths = @(
  "$env:ProgramFiles\Microsoft OneDrive\OneDrive.exe",
  "${env:ProgramFiles(x86)}\Microsoft OneDrive\OneDrive.exe"
)

$timeoutSeconds = 240
$deadline = (Get-Date).AddSeconds($timeoutSeconds)
$installed = $false

while ((Get-Date) -lt $deadline) {
  $exeExists = $false
  foreach ($p in $exePaths) {
    if ($p -and (Test-Path $p)) { $exeExists = $true; break }
  }
  if ((Test-Path $uninstallKey) -or $exeExists) {
    $installed = $true
    break
  }
  # If the top-level setup process exited, capture its code and stop polling.
  if ($process.HasExited) { break }
  Start-Sleep -Seconds 5
}

# Final check in case the setup process exited right before the loop bailed.
if (-not $installed) {
  $exeExists = $false
  foreach ($p in $exePaths) {
    if ($p -and (Test-Path $p)) { $exeExists = $true; break }
  }
  if ((Test-Path $uninstallKey) -or $exeExists) { $installed = $true }
}

if ($installed) {
  Write-Host "OneDrive per-machine install detected."
  Exit 0
}

if ($process.HasExited) {
  $exitCode = $process.ExitCode
  Write-Host "OneDriveSetup exited with code: $exitCode"
  if ($exitCode -eq 0 -or $exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
  Exit $exitCode
}

Write-Host "Timed out waiting for OneDrive install to complete."
Exit 1

} catch {
  Write-Host "Error: $_"
  Exit 1
}
