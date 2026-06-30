# Attempts to locate ActivityWatch's Inno Setup uninstaller from the registry and run it silently.

$displayNameLike = "ActivityWatch*"
$publisher = "ActivityWatch Contributors"

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

$uninstallCommand = $uninstall.UninstallString
$uninstallArgs = "/VERYSILENT /NORESTART"

if ($uninstallCommand -match '^"([^"]+)"\s*(.*)$') {
    $uninstallCommand = $matches[1]; $extra = $matches[2].Trim()
    if ($extra) { $uninstallArgs = "$extra $uninstallArgs".Trim() }
} elseif ($uninstallCommand -match '^(.+?\.exe)\s*(.*)$') {
    $uninstallCommand = $matches[1]; $extra = $matches[2].Trim()
    if ($extra) { $uninstallArgs = "$extra $uninstallArgs".Trim() }
} else {
    Write-Host "Error: Could not parse uninstall command: $uninstallCommand"; Exit 1
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
