$displayNameLike = "Lenovo Dock Manager*"
$publisherLike = "Lenovo*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$ExpectedExitCodes = @(0, 1641, 3010, 1223)
# Give the uninstaller close to the full validation window; with the components
# stopped it completes well under this, but it legitimately needs more than a
# couple of minutes.
$timeoutSeconds = 300

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

if (-not $entry -or -not $entry.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

# Close the Dock Manager components first (the service plus the dockmgr,
# dockmgr.schd and dockmgr.svc processes). The Inno uninstaller relaunches to a
# temp copy and the parent waits on it, and that temp uninstaller blocks waiting
# on the running service/app under the SYSTEM context. Uses only built-in
# PowerShell cmdlets (no external toolkit required).
Get-CimInstance Win32_Service -ErrorAction SilentlyContinue | Where-Object {
  $_.PathName -like '*dockmgr*' -or $_.PathName -like '*Dock Manager*'
} | ForEach-Object { Stop-Service -Name $_.Name -Force -ErrorAction SilentlyContinue }
foreach ($n in @('dockmgr', 'dockmgr.schd', 'dockmgr.svc')) {
  Stop-Process -Name $n -Force -ErrorAction SilentlyContinue
}

# Inno's UninstallString is the bare unins000.exe path (quoted, no args). Run it
# with /VERYSILENT /NORESTART (not /SILENT, which renders a progress window that
# stalls in the SYSTEM/session-0 context).
$uninstPath = ($entry.UninstallString -replace '"', '').Trim()

if (-not (Test-Path -LiteralPath $uninstPath)) {
  Write-Host "Uninstaller not found at: $uninstPath"
  Exit 1
}

Write-Host "Uninstall command: $uninstPath"
Write-Host "Uninstall args: /VERYSILENT /NORESTART"

try {
    Start-Process -FilePath $uninstPath -ArgumentList "/VERYSILENT /NORESTART" | Out-Null

    # The original unins000.exe relaunches to a temp copy and may return before the
    # actual removal finishes, so poll the registry (the true completion signal)
    # rather than waiting on a single process. The uninstaller removes its entry
    # when it completes.
    $elapsed = 0
    while ($elapsed -lt $timeoutSeconds) {
        if (-not (Test-StillInstalled)) { break }
        Start-Sleep -Seconds 5
        $elapsed += 5
    }
    Start-Sleep -Seconds 5

    if (-not (Test-StillInstalled)) {
        Write-Host "Dock Manager uninstalled successfully."
        Exit 0
    }

    # Still present after the timeout: stop any lingering uninstaller (original
    # plus the temp _iu*.tmp copy) and re-check.
    Stop-Process -Name "unins000" -Force -ErrorAction SilentlyContinue
    Get-Process -ErrorAction SilentlyContinue | Where-Object { $_.Name -like "_iu*" } |
      Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 5

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
