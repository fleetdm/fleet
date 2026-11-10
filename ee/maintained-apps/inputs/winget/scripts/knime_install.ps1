# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# KNIME Analytics Platform ships as an Inno Setup-based installer.
# Machine-scope install uses /ALLUSERS.
#
# Setup won't install over an existing KNIME: it asks to uninstall it first, and
# /SUPPRESSMSGBOXES doesn't answer that prompt, so setup waits forever. Remove the
# existing install silently first. Unsaved work in an open KNIME is lost.

$exeFilePath = "${env:INSTALLER_PATH}"
$displayName = "KNIME Analytics Platform"
$publisher = "KNIME AG"

function Get-KnimeEntry {
  $paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
  )
  foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
      $_.DisplayName -eq $displayName -and $_.Publisher -eq $publisher
    }
    if ($items) { return $items | Select-Object -First 1 }
  }
  return $null
}

function Remove-ExistingKnime {
  $entry = Get-KnimeEntry
  if (-not $entry) { return }
  if (-not $entry.UninstallString) {
    Throw "Found $displayName $($entry.DisplayVersion), but it has no uninstall command. Uninstall it, then install again."
  }
  Write-Host "Removing existing install: $displayName $($entry.DisplayVersion)"

  Stop-Process -Name "knime" -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 2

  $uninstallCommand = $entry.UninstallString
  if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]; $uninstallArgs = $matches[2].Trim()
  } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]; $uninstallArgs = $matches[2].Trim()
  } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]; $uninstallArgs = $matches[2].Trim()
  } else {
    Throw "Could not parse uninstall command: $uninstallCommand"
  }
  $uninstallArgs = "$uninstallArgs /VERYSILENT /SUPPRESSMSGBOXES /NORESTART".Trim()

  Write-Host "Uninstall command: $exePath $uninstallArgs"
  $process = Start-Process -FilePath $exePath -ArgumentList $uninstallArgs -NoNewWindow -PassThru -Wait
  Write-Host "Uninstall exit code: $($process.ExitCode)"
  if ($process.ExitCode -ne 0) {
    Throw "Uninstalling $displayName $($entry.DisplayVersion) failed with exit code $($process.ExitCode)."
  }

  # unins000.exe hands off to a temporary copy of itself (_iu*.tmp) and exits
  # before the removal finishes.
  $deadline = (Get-Date).AddMinutes(5)
  do {
    Start-Sleep -Seconds 2
    $pending = (Get-KnimeEntry) -or (Get-Process -ErrorAction SilentlyContinue |
      Where-Object { $_.Path -and (Split-Path $_.Path -Leaf) -like '_iu*.tmp' })
  } while ($pending -and (Get-Date) -lt $deadline)
  if ($pending) {
    Throw "Timed out removing $displayName $($entry.DisplayVersion)."
  }
}

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

Remove-ExistingKnime

# Inno Setup silent install for all users (machine scope)
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /ALLUSERS"
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
