# A silent uninstall also deletes Mullvad VPN's settings and removes this
# device from the Mullvad account.
#
# The uninstaller is run from a temporary copy with the NSIS "_?=" flag, so
# that it runs to completion and can delete its own install directory.

$uninstallKeyName = '2A356FD4-03B7-4F45-99B4-737BE580DC82'
$timeoutSeconds = 600

$paths = @(
  "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\$uninstallKeyName",
  "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\$uninstallKeyName"
)

function Find-UninstallEntry {
  foreach ($p in $paths) {
    $item = Get-ItemProperty -LiteralPath $p -ErrorAction SilentlyContinue
    if ($item -and $item.DisplayName -like 'Mullvad VPN*') { return $item }
  }
  return $null
}

$entry = Find-UninstallEntry
if (-not $entry -or (-not $entry.QuietUninstallString -and -not $entry.UninstallString)) {
  Write-Host "Mullvad VPN uninstall entry not found; nothing to do."
  Exit 0
}

$uninstallCommand = if ($entry.QuietUninstallString) {
  $entry.QuietUninstallString
} else {
  $entry.UninstallString
}

$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
  $exePath = $matches[1]
  $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
  $exePath = $matches[1]
  $existingArgs = $matches[2].Trim()
} else {
  Write-Host "Could not parse uninstall string: $uninstallCommand"
  Exit 1
}

if (-not (Test-Path -LiteralPath $exePath -PathType Leaf)) {
  Write-Host "Uninstaller not found: $exePath"
  Exit 1
}

if ($existingArgs -notmatch '(?i)(^|\s)/S(\s|$)') {
  $existingArgs = ("$existingArgs /S").Trim()
}

# "_?=" must be the last argument and its path must not be quoted.
$installDir = (Split-Path -Parent $exePath).TrimEnd('\')
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("fleet-mullvad-" + [guid]::NewGuid().ToString())
$tempExe = Join-Path $tempDir "uninstall-mullvad-vpn.exe"

try {
  New-Item -ItemType Directory -Path $tempDir -ErrorAction Stop | Out-Null
  Copy-Item -LiteralPath $exePath -Destination $tempExe -ErrorAction Stop

  $uninstallArgs = "$existingArgs _?=$installDir"
  Write-Host "Uninstall command: $tempExe"
  Write-Host "Uninstall args: $uninstallArgs"

  $process = Start-Process -FilePath $tempExe -ArgumentList $uninstallArgs -PassThru -NoNewWindow
  # Touch the handle so ExitCode is still readable after the process exits.
  try { $null = $process.Handle } catch { }

  if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
    Write-Host "Uninstaller did not exit within $timeoutSeconds seconds; terminating it."
    & taskkill.exe /PID $process.Id /T /F 2>&1 | Write-Host
    Exit 1603
  }

  $exitCode = $process.ExitCode
  Write-Host "Uninstall exit code: $exitCode"
  if ($exitCode -ne 0) {
    Exit $exitCode
  }

  if (Find-UninstallEntry) {
    Write-Host "Mullvad VPN is still registered after the uninstaller exited."
    Exit 1
  }

  Write-Host "Mullvad VPN is no longer present."
  Exit 0
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
} finally {
  # A failed uninstall can delete the uninstaller but stay registered. Put it
  # back so the uninstall can be retried.
  if ((Find-UninstallEntry) -and (Test-Path -LiteralPath $tempExe -PathType Leaf) -and
      -not (Test-Path -LiteralPath $exePath -PathType Leaf)) {
    New-Item -ItemType Directory -Path $installDir -Force -ErrorAction SilentlyContinue | Out-Null
    Copy-Item -LiteralPath $tempExe -Destination $exePath -ErrorAction SilentlyContinue
    if (Test-Path -LiteralPath $exePath -PathType Leaf) {
      Write-Host "Restored the uninstaller to $exePath."
    }
  }
  Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
