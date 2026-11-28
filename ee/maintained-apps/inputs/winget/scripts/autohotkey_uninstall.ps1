# Removes the all-users install of AutoHotkey v2. Running AutoHotkey scripts
# and open AutoHotkey help windows are stopped first: the uninstaller can't
# delete files in use and reports that in a dialog, even with /silent.
# Files changed since they were installed are left in place by the uninstaller.
# A per-user install ("AutoHotkey (user)") or a v1-only install is not touched.

$timeoutSeconds = 300
$uninstallKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\AutoHotkey'

function Get-UninstallEntry {
  $entry = Get-ItemProperty -LiteralPath $uninstallKey -ErrorAction SilentlyContinue
  if ($entry -and $entry.DisplayName -eq 'AutoHotkey') { return $entry }
  return $null
}

function Remove-DirIfEmpty {
  param([string]$dir)
  if (-not $dir -or -not (Test-Path -LiteralPath $dir -PathType Container)) { return }
  if (@(Get-ChildItem -LiteralPath $dir -Force -ErrorAction SilentlyContinue).Count -eq 0) {
    Remove-Item -LiteralPath $dir -Force -ErrorAction SilentlyContinue
  }
}

$entry = Get-UninstallEntry
if (-not $entry) {
  Write-Host "AutoHotkey is not installed for all users; nothing to do."
  Exit 0
}

try {
  $command = $entry.QuietUninstallString
  if (-not $command) {
    Write-Host "AutoHotkey has no QuietUninstallString; can't uninstall silently."
    Exit 1
  }

  if ($command -match '^\s*"([^"]+)"\s*(.*)$') {
    $uninstallExe = $matches[1]
    $uninstallArgs = $matches[2]
  } elseif ($command -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $uninstallExe = $matches[1]
    $uninstallArgs = $matches[2]
  } else {
    Write-Host "Can't parse QuietUninstallString: $command"
    Exit 1
  }
  # Without /silent the uninstaller shows dialogs that nobody can answer.
  if ($uninstallArgs -notmatch '(?i)(^|\s)/silent($|\s)') {
    Write-Host "QuietUninstallString is not a silent AutoHotkey uninstall: $command"
    Exit 1
  }

  $installDir = [string]$entry.InstallLocation
  if (-not $installDir) { $installDir = Split-Path -Parent (Split-Path -Parent $uninstallExe) }
  $installDir = $installDir.TrimEnd('\')
  $prefix = "$installDir\"

  Get-CimInstance Win32_Process -ErrorAction SilentlyContinue | Where-Object {
    ($_.Name -like 'AutoHotkey*' -and $_.ExecutablePath -and $_.ExecutablePath.StartsWith($prefix, [StringComparison]::OrdinalIgnoreCase)) -or
    ($_.Name -eq 'hh.exe' -and $_.CommandLine -and
      $_.CommandLine.IndexOf($prefix, [StringComparison]::OrdinalIgnoreCase) -ge 0)
  } | ForEach-Object {
    Write-Host "Stopping $($_.Name) (PID $($_.ProcessId))"
    Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
  }
  Start-Sleep -Seconds 2

  Write-Host "Uninstall command: `"$uninstallExe`" $uninstallArgs"
  $process = Start-Process -FilePath $uninstallExe -ArgumentList $uninstallArgs -PassThru
  # Touch the handle so ExitCode is still readable after the process exits.
  try { $null = $process.Handle } catch { }
  if ($process.WaitForExit($timeoutSeconds * 1000)) {
    Write-Host "Uninstall exit code: $($process.ExitCode)"
  } else {
    Write-Host "The AutoHotkey uninstaller did not finish within $timeoutSeconds seconds. It is probably waiting on an error dialog; stopping it."
    & taskkill.exe /PID $process.Id /T /F 2>&1 | Write-Host
  }

  if (Get-UninstallEntry) {
    Write-Host "AutoHotkey is still registered after the uninstaller ran."
    Exit 1
  }

  # The uninstaller runs from UX\AutoHotkeyUX.exe and removes that file and
  # the then-empty folders a moment after it exits. Wait, then finish the job.
  $uxExe = Join-Path $installDir 'UX\AutoHotkeyUX.exe'
  $stopwatch = [Diagnostics.Stopwatch]::StartNew()
  while ((Test-Path -LiteralPath $uxExe) -and $stopwatch.Elapsed.TotalSeconds -lt 15) {
    Start-Sleep -Seconds 1
  }
  Remove-Item -LiteralPath $uxExe -Force -ErrorAction SilentlyContinue
  Remove-DirIfEmpty (Join-Path $installDir 'UX')
  Remove-DirIfEmpty $installDir

  if (Test-Path -LiteralPath $installDir) {
    Write-Host "AutoHotkey was uninstalled. Files that were not part of the install or were modified remain in $installDir."
  } else {
    Write-Host "AutoHotkey was uninstalled."
  }
  Exit 0
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
