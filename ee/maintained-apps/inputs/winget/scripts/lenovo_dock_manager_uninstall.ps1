$displayNameLike = "Lenovo Dock Manager*"
$publisherLike = "Lenovo*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$ExpectedExitCodes = @(0, 1641, 3010, 1223)
$timeoutSeconds = 180

function Test-StillInstalled {
  foreach ($p in $paths) {
    $m = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
      $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike
    }
    if ($m) { return $true }
  }
  return $false
}

$entry = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike
  }
  if ($items) { $entry = $items | Select-Object -First 1; break }
}

if (-not $entry -or (-not $entry.UninstallString -and -not $entry.QuietUninstallString)) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

# Stop the Dock Manager service and processes first. The Inno uninstaller
# relaunches itself to a temp copy and the parent waits on it; that temp
# uninstaller blocks indefinitely waiting on the running service/app under the
# SYSTEM context, so we shut them down before uninstalling.
Get-Service -ErrorAction SilentlyContinue | Where-Object {
  $_.Name -like "*DockMgr*" -or $_.Name -like "*DockManager*" -or $_.DisplayName -like "*Dock Manager*"
} | Stop-Service -Force -ErrorAction SilentlyContinue
Stop-Process -Name "dockmgr" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "dockmgr.svc" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "dockmgr.schd" -Force -ErrorAction SilentlyContinue

$uninstallCommand = if ($entry.QuietUninstallString) { $entry.QuietUninstallString } else { $entry.UninstallString }

$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]; $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]; $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]; $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

if ($existingArgs -notmatch '(?i)/VERYSILENT') { $existingArgs = ("$existingArgs /VERYSILENT").Trim() }
if ($existingArgs -notmatch '(?i)/SUPPRESSMSGBOXES') { $existingArgs = ("$existingArgs /SUPPRESSMSGBOXES").Trim() }
if ($existingArgs -notmatch '(?i)/NORESTART') { $existingArgs = ("$existingArgs /NORESTART").Trim() }

Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

try {
    $process = Start-Process -FilePath $exePath -ArgumentList $existingArgs -NoNewWindow -PassThru

    # The Inno uninstaller removes its registry entry when it completes. Poll for
    # that instead of blocking on the process, which relaunches to a temp copy.
    $elapsed = 0
    while ($elapsed -lt $timeoutSeconds) {
        if (-not (Test-StillInstalled)) { break }
        Start-Sleep -Seconds 3
        $elapsed += 3
    }

    if (Test-StillInstalled) {
        # Still present after the timeout: stop any lingering uninstaller processes.
        Stop-Process -Name "unins000" -Force -ErrorAction SilentlyContinue
        if (-not $process.HasExited) {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
    }

    if (-not (Test-StillInstalled)) {
        Write-Host "Dock Manager uninstalled successfully."
        Exit 0
    }

    Write-Host "Dock Manager still present after uninstall attempt."
    Exit 1
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
