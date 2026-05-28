# Attempts to locate Evernote's NSIS uninstaller from registry and execute it silently.

$displayNameLike = "Evernote*"
$publisher = "Evernote Corporation"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and $_.Publisher -like "$publisher*"
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

Stop-Process -Name "Evernote" -Force -ErrorAction SilentlyContinue

$uninstallCommand = $uninstall.UninstallString

$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

# Per silentinstallhq the documented silent uninstall flags are "/AllUsers /S"
# for Evernote 10.x+ installs.
if ($existingArgs -notmatch '/AllUsers') {
    $existingArgs = ("/AllUsers $existingArgs").Trim()
}
if ($existingArgs -notmatch '\b/S\b') {
    $existingArgs = ("$existingArgs /S").Trim()
}

# NSIS uninstallers copy themselves to %TEMP%\Au_.exe and exit the original
# process immediately, so `Start-Process -Wait` returns in ~1s while the real
# uninstall is still running. Two mitigations:
#  1. Pass `_?=<install_dir>` (documented NSIS option) to disable the
#     self-copy and keep the original process running synchronously.
#  2. Wait afterward for any leftover Au_*.exe / Un_*.exe helpers to exit.
$installDir = Split-Path -Path $exePath -Parent
$existingArgs = "$existingArgs _?=`"$installDir`""

Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

try {
    $processOptions = @{
        FilePath = $exePath
        ArgumentList = $existingArgs
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    # Fallback: wait for any NSIS helper processes that the uninstaller may
    # have spawned regardless of `_?=`. Cap at 5 minutes.
    $deadline = (Get-Date).AddMinutes(5)
    while ((Get-Date) -lt $deadline) {
        $helpers = Get-Process -ErrorAction SilentlyContinue |
            Where-Object { $_.Name -match '^(Au_|Un_).*' -or $_.Path -like "*\Evernote\*" }
        if (-not $helpers) { break }
        Start-Sleep -Seconds 2
    }

    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
