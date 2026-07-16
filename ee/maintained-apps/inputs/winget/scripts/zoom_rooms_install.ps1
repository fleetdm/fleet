# Zoom Rooms ships an MSI bootstrapper: msiinfo shows the MSI's
# ProductName = "Zoom Rooms Installer" and its CustomAction.Install execs an
# embedded EXE with `--accept_gdpr=[ACCEPTGDPR] --silent=[SILENT] ...`. The
# embedded EXE installs the *real* Zoom Rooms app at C:\Program Files\ZoomRooms
# and registers a separate ARP entry. Without ACCEPTGDPR=true and SILENT=true
# the bootstrapper EXE waits on a consent prompt and the validator kills it at
# the 5-minute install timeout (observed in CI). The flag set below is taken
# verbatim from the winget manifest (Zoom.ZoomRooms InstallerSwitches:
# Silent + Custom).

$msiFilePath = "${env:INSTALLER_PATH}"

try {

  if (-not (Test-Path $msiFilePath)) {
    Write-Host "Error: Installer file not found at: $msiFilePath"
    Exit 1
  }

  $argumentList = @(
    "/i", "`"$msiFilePath`"",
    "/passive",
    "/norestart",
    "ACCEPTGDPR=true",
    "SILENT=true",
    "AUTOSTART=false",
    "ZLAUNCHAPP=0"
  )

  Write-Host "Install command: msiexec.exe $($argumentList -join ' ')"

  $processOptions = @{
    FilePath     = "msiexec.exe"
    ArgumentList = $argumentList
    PassThru     = $true
    Wait         = $true
  }

  $process = Start-Process @processOptions
  $exitCode = $process.ExitCode
  Write-Host "Install exit code: $exitCode"

  # MSI success codes: 0 = success, 3010 = reboot required, 1641 = reboot initiated.
  if ($exitCode -in 0, 3010, 1641) { Exit 0 }
  Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
