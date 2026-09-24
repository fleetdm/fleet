# AnyBurn ships an NSIS uninstaller. "uninstall.exe /S" with Start-Process -Wait
# hung until the validator's 10-minute timeout: -Wait also waits on descendants,
# and without the NSIS "_?=" flag the uninstaller relaunches itself from %TEMP%.
# So run it in place, wait on that process only, and finish the job ourselves —
# "_?=" leaves the uninstaller and its directory behind even on success.

$displayName = "AnyBurn"
$processName = "anyburn"
$timeoutSeconds = 120

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

# Exact DisplayName match so "AnyBurn Pro" is left alone.
function Find-UninstallEntry {
  foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
      $_.DisplayName -eq $displayName
    }
    if ($items) { return $items | Select-Object -First 1 }
  }
  return $null
}

function Remove-InstallDir {
  param([string]$dir)
  if (-not $dir) { return }
  $resolved = $null
  try { $resolved = (Resolve-Path -LiteralPath $dir -ErrorAction Stop).Path } catch { return }
  # Never recurse a drive root or a two-segment path like C:\Windows.
  if (($resolved -match '^[A-Za-z]:\\') -and ((($resolved.TrimEnd('\')) -split '\\').Count -ge 3)) {
    Remove-Item -LiteralPath $resolved -Recurse -Force -ErrorAction SilentlyContinue
  }
}

$entry = Find-UninstallEntry
if (-not $entry -or -not $entry.UninstallString) {
  Write-Host "Uninstall entry for '$displayName' not found; nothing to do."
  Exit 0
}

try {
  $uninstallString = $entry.UninstallString
  if ($uninstallString -match '^"([^"]+)"') {
    $uninstallExe = $matches[1]
  } elseif ($uninstallString -match '^(.+?\.exe)') {
    $uninstallExe = $matches[1]
  } else {
    $uninstallExe = $uninstallString
  }

  $installDir = $entry.InstallLocation
  if (-not $installDir -or -not (Test-Path -LiteralPath $installDir)) {
    $installDir = Split-Path -Parent $uninstallExe
  }
  # A quoted argument ending in "\" would escape its own closing quote.
  if ($installDir) { $installDir = $installDir.TrimEnd('\') }

  Stop-Process -Name $processName -Force -ErrorAction SilentlyContinue

  $uninstallArgs = @("/S", "_?=$installDir")
  Write-Host "Uninstall command: $uninstallExe"
  Write-Host "Uninstall args: $uninstallArgs"

  $process = Start-Process -FilePath $uninstallExe -ArgumentList $uninstallArgs `
    -PassThru -NoNewWindow
  # Touch the handle so ExitCode is still readable after the process exits.
  try { $null = $process.Handle } catch { }
  if ($process.WaitForExit($timeoutSeconds * 1000)) {
    Write-Host "Uninstall exit code: $($process.ExitCode)"
  } else {
    Write-Host "Uninstaller did not exit within $timeoutSeconds seconds; terminating it."
    & taskkill.exe /PID $process.Id /T /F 2>&1 | Write-Host
  }

  Stop-Process -Name "Au_" -Force -ErrorAction SilentlyContinue

  # Don't trust the uninstaller's outcome; check what is actually left.
  $remaining = Find-UninstallEntry
  if ($remaining) {
    Write-Host "'$displayName' is still registered; removing it manually."
    Remove-Item -LiteralPath $remaining.PSPath -Recurse -Force -ErrorAction SilentlyContinue
  }

  $shortcuts = @(
    "$env:ProgramData\Microsoft\Windows\Start Menu\Programs\$displayName",
    "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\$displayName",
    "$env:PUBLIC\Desktop\$displayName.lnk",
    "$env:USERPROFILE\Desktop\$displayName.lnk"
  )
  foreach ($shortcut in $shortcuts) {
    Remove-Item -LiteralPath $shortcut -Recurse -Force -ErrorAction SilentlyContinue
  }

  Remove-InstallDir $installDir

  if (Find-UninstallEntry) {
    Write-Host "'$displayName' is still present after removal attempts."
    Exit 1
  }

  Write-Host "'$displayName' is no longer present."
  Exit 0
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
