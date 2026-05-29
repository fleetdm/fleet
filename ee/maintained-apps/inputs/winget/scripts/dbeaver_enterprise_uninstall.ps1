# Locates DBeaverEE's NSIS uninstaller from the registry and runs it silently.
# DBeaver NSIS DisplayName embeds the version (e.g. "DBeaverEE 26.0.0"), so
# match by prefix. Per skill pitfall #14, also use NSIS _?= to disable the
# self-fork so Start-Process -Wait actually waits for the uninstall to finish.

$displayNameLike = "DBeaverEE*"
$publisher = "DBeaver Corp"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and $_.Publisher -eq $publisher
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

Stop-Process -Name "dbeaver" -Force -ErrorAction SilentlyContinue

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

if ($existingArgs -notmatch '\b/S\b') {
    $existingArgs = ("$existingArgs /S").Trim()
}

# NOTE: do NOT add `_?=<install_dir>` here. DBeaver's NSIS uninstaller
# returns exit 2 ("user cancelled") when invoked with `_?=`. Per
# silentinstallhq.com/dbeaver-silent-install-how-to-guide the documented
# silent uninstall is just `Uninstall.exe /allusers /S` — the existing
# DBeaver Community FMA also uses this form successfully.

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
    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
