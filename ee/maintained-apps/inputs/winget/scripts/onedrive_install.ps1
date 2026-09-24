# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# OneDriveSetup.exe performs a per-machine install with "/allusers /silent"
# (switches verified against the winget InstallerSwitches Custom: /allusers and
# silentinstallhq.com). The catch: OneDriveSetup.exe spawns several child
# processes and starts the resident OneDrive.exe, so a plain Start-Process -Wait
# can wait indefinitely and hit the CI step timeout. Instead, start the
# installer and poll for the per-machine ARP (uninstall) registry entry.
#
# The ARP entry is the completion signal — not files on disk. The installer
# drops OneDrive.exe under Program Files well before it registers the
# uninstall key, and the uninstall key (DisplayName/DisplayVersion) is what
# osquery's programs table — and therefore Fleet's detection — reads. Waiting
# on the binary races detection. Modern x64 OneDrive registers in the native
# hive; older builds used WOW6432Node, so check both. Registration is done by
# child processes, so keep polling to the deadline even after the top-level
# setup process exits.

$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/allusers /silent" -PassThru

$uninstallKeys = @(
  "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\OneDriveSetup.exe",
  "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\OneDriveSetup.exe"
)

function Test-OneDriveRegistered {
  foreach ($key in $uninstallKeys) {
    if (Test-Path $key) {
      $entry = Get-ItemProperty -Path $key -ErrorAction SilentlyContinue
      if ($entry -and $entry.DisplayName -and $entry.DisplayVersion) {
        return $true
      }
    }
  }
  return $false
}

$timeoutSeconds = 240
$deadline = (Get-Date).AddSeconds($timeoutSeconds)
$installed = $false

while ((Get-Date) -lt $deadline) {
  if (Test-OneDriveRegistered) {
    $installed = $true
    break
  }
  Start-Sleep -Seconds 5
}

if ($installed) {
  Write-Host "OneDrive per-machine install registered."
  # Give the setup process a moment to exit so the installer file isn't locked.
  if (-not $process.HasExited) {
    Wait-Process -Id $process.Id -Timeout 60 -ErrorAction SilentlyContinue
  }
  Exit 0
}

if ($process.HasExited) {
  $exitCode = $process.ExitCode
  Write-Host "OneDriveSetup exited with code: $exitCode, but no per-machine registry entry was found."
  if ($exitCode -ne 0 -and $exitCode -ne 3010 -and $exitCode -ne 1641) { Exit $exitCode }
  Exit 1
}

Write-Host "Timed out waiting for OneDrive install to complete."
Exit 1

} catch {
  Write-Host "Error: $_"
  Exit 1
}
