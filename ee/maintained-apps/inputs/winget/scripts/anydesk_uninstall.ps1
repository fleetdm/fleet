# Attempts to locate AnyDesk's uninstaller from registry and execute it silently.
# AnyDesk supports --remove --silent.

$displayName = "AnyDesk"
$publisher = "AnyDesk Software GmbH"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -eq $displayName -and $_.Publisher -like "$publisher*"
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

Stop-Process -Name "AnyDesk" -Force -ErrorAction SilentlyContinue

$uninstallCommand = $uninstall.UninstallString

$exePath = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

$existingArgs = "--remove --silent"

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
