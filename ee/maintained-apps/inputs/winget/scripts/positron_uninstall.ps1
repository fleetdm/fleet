# Removes the machine-wide Positron install. A per-user install registers as
# "Positron (User)" and is left alone.
$displayName = "Positron"
$publisher = "Posit Software, PBC"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -eq $displayName -and $_.Publisher -eq $publisher
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString)) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

$uninstallCommand = if ($uninstall.QuietUninstallString) {
    $uninstall.QuietUninstallString
} else {
    $uninstall.UninstallString
}

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

foreach ($flag in @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART')) {
    if ($existingArgs -notmatch [regex]::Escape($flag)) {
        $existingArgs = ("$existingArgs $flag").Trim()
    }
}

# Positron is stopped before its files are removed. Unsaved changes are lost.
$installDir = if ($uninstall.InstallLocation) {
    "$($uninstall.InstallLocation)".TrimEnd('\')
} else {
    Split-Path -Parent $exePath
}
$running = @(Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) })
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before uninstalling."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

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
