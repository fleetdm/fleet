# Windscribe's uninstaller is silent only with /VERYSILENT. The /SILENT switch
# in its registered QuietUninstallString still shows a confirmation prompt.
#
# The uninstaller copies itself to a temporary directory, starts that copy and
# exits, so this script waits for the copy and then checks the result itself.

$uninstallKeyName = '{fa690e90-ddb0-4f0c-b3f1-136c084e5fc7}_is1'
$timeoutSeconds = 600

$paths = @(
  "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\$uninstallKeyName",
  "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\$uninstallKeyName"
)

function Find-UninstallEntry {
  foreach ($p in $paths) {
    $item = Get-ItemProperty -LiteralPath $p -ErrorAction SilentlyContinue
    if ($item -and $item.DisplayName -eq 'Windscribe') { return $item }
  }
  return $null
}

function Get-RelaunchedUninstaller {
  Get-Process -Name "uninstall" -ErrorAction SilentlyContinue | Where-Object {
    $_.Path -like "$env:WINDIR\Temp\WindscribeUninstaller*"
  }
}

$entry = Find-UninstallEntry
if (-not $entry -or -not $entry.UninstallString) {
  Write-Host "Windscribe uninstall entry not found; nothing to do."
  Exit 0
}

$uninstallCommand = $entry.UninstallString
$exePath = ""
if ($uninstallCommand -match '^\s*"([^"]+)"') {
  $exePath = $matches[1]
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)') {
  $exePath = $matches[1]
} else {
  Write-Host "Could not parse uninstall string: $uninstallCommand"
  Exit 1
}

if (-not (Test-Path -LiteralPath $exePath -PathType Leaf)) {
  Write-Host "Uninstaller not found: $exePath"
  Exit 1
}

try {
  # A silent uninstall does not ask the user to close Windscribe first.
  Stop-Process -Name "Windscribe" -Force -ErrorAction SilentlyContinue

  Write-Host "Uninstall command: $exePath /VERYSILENT"
  $process = Start-Process -FilePath $exePath -ArgumentList "/VERYSILENT" -PassThru -NoNewWindow
  # Touch the handle so ExitCode is still readable after the process exits.
  try { $null = $process.Handle } catch { }

  if (-not $process.WaitForExit(60 * 1000)) {
    Write-Host "Uninstaller did not hand off within 60 seconds; terminating it."
    & taskkill.exe /PID $process.Id /T /F 2>&1 | Write-Host
    Exit 1603
  }

  # On success the exit code is the process ID of the relaunched copy; -1 means
  # the uninstaller could not start.
  if ($process.ExitCode -eq -1) {
    Write-Host "Uninstaller failed to start."
    Exit 1
  }

  $deadline = (Get-Date).AddSeconds($timeoutSeconds)
  while ((Get-RelaunchedUninstaller) -and ((Get-Date) -lt $deadline)) {
    Start-Sleep -Seconds 2
  }

  $stillRunning = Get-RelaunchedUninstaller
  if ($stillRunning) {
    Write-Host "Uninstaller did not finish within $timeoutSeconds seconds; terminating it."
    $stillRunning | Stop-Process -Force -ErrorAction SilentlyContinue
    Exit 1603
  }

  if (Find-UninstallEntry) {
    Write-Host "Windscribe is still registered after the uninstaller exited."
    Exit 1
  }

  Write-Host "Windscribe is no longer present."
  Exit 0
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}
