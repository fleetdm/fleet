# Uninstalling closes the Sandboxie-Plus window, ends every sandboxed program,
# and removes the Sandboxie driver and service. Sandboxes and settings
# (Sandboxie.ini) are kept.

$uninstallKeyName = 'Sandboxie-Plus_is1'
$timeoutSeconds = 480

$paths = @(
  "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\$uninstallKeyName",
  "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\$uninstallKeyName"
)

function Find-UninstallEntry {
  foreach ($p in $paths) {
    $item = Get-ItemProperty -LiteralPath $p -ErrorAction SilentlyContinue
    if ($item -and $item.DisplayName -like 'Sandboxie-Plus*') { return $item }
  }
  return $null
}

$entry = Find-UninstallEntry
if (-not $entry -or -not $entry.UninstallString) {
  Write-Host "Sandboxie-Plus uninstall entry not found; nothing to do."
  Exit 0
}

$exePath = ""
if ($entry.UninstallString -match '^\s*"([^"]+)"') {
  $exePath = $matches[1]
} elseif ($entry.UninstallString -match '(?i)^\s*(.+?\.exe)(\s|$)') {
  $exePath = $matches[1]
} else {
  Write-Host "Could not parse uninstall string: $($entry.UninstallString)"
  Exit 1
}

if (-not (Test-Path -LiteralPath $exePath -PathType Leaf)) {
  Write-Host "Uninstaller not found: $exePath"
  Exit 1
}

try {
  $uninstallArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
  Write-Host "Uninstall command: $exePath"
  Write-Host "Uninstall args: $uninstallArgs"

  $process = Start-Process -FilePath $exePath -ArgumentList $uninstallArgs -PassThru
  # Keeps .ExitCode readable after the process ends.
  $null = $process.Handle

  # A driver or service error shows a message box even when silent.
  if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
    Write-Host "Uninstaller did not exit within $timeoutSeconds seconds; terminating it."
    & taskkill.exe /PID $process.Id /T /F 2>&1 | Write-Host
    Exit 1603
  }

  $exitCode = $process.ExitCode
  Write-Host "Uninstall exit code: $exitCode"

  # 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
  if ($exitCode -ne 0 -and $exitCode -ne 3010 -and $exitCode -ne 1641) {
    Exit $exitCode
  }

  if (Find-UninstallEntry) {
    Write-Host "Sandboxie-Plus is still registered after the uninstaller exited."
    Exit 1
  }

  Write-Host "Sandboxie-Plus is no longer present."
  Exit 0
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
