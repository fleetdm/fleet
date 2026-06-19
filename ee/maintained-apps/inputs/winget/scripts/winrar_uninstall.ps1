# Attempts to locate WinRAR's uninstaller from the registry and run it silently.
# WinRAR's UninstallString points at "<install dir>\uninstall.exe", which accepts /S.

$displayNameLike = "WinRAR*"
$publisher = "win.rar GmbH"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
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

Stop-Process -Name "WinRAR" -Force -ErrorAction SilentlyContinue

$uninstallCommand = $uninstall.UninstallString
$uninstallArgs = "/S"

# Parse the UninstallString defensively for the three common shapes.
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {            # quoted path
    $uninstallCommand = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {  # unquoted, may contain spaces
    $uninstallCommand = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {           # bare token
    $uninstallCommand = $matches[1]
    $existingArgs = $matches[2].Trim()
}

if ($existingArgs -and $existingArgs -ne '') {
    $uninstallArgs = "$existingArgs $uninstallArgs"
}

Write-Host "Uninstall command: $uninstallCommand"
Write-Host "Uninstall args: $uninstallArgs"

try {
    $processOptions = @{
        FilePath = $uninstallCommand
        ArgumentList = $uninstallArgs
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
