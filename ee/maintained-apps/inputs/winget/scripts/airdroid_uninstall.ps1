# Attempts to locate AirDroid's NSIS uninstaller from the registry and run it silently.
# The registry DisplayName includes the version (e.g. "AirDroid 3.8.0.4"), so match a prefix.

$displayNameLike = "AirDroid*"
$publisher = "AirDroid"

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
$uninstallArgs = "/S"

$splitArgs = $uninstallCommand.Split('"')
if ($splitArgs.Length -gt 1) {
    if ($splitArgs.Length -eq 3) {
        $existingArgs = $splitArgs[2].Trim()
        if ($existingArgs -ne '' -and $existingArgs -notmatch '\b/S\b') {
            $uninstallArgs = "$existingArgs /S".Trim()
        } elseif ($existingArgs -ne '') {
            $uninstallArgs = $existingArgs
        }
    } elseif ($splitArgs.Length -gt 3) {
        Write-Host "Error: Uninstall command contains multiple quoted strings"
        Exit 1
    }
    $uninstallCommand = $splitArgs[1]
} else {
    if ($uninstallCommand -notmatch '\b/S\b') { $uninstallArgs = "/S" } else { $uninstallArgs = "" }
}

Write-Host "Uninstall command: $uninstallCommand"
Write-Host "Uninstall args: $uninstallArgs"

try {
    $processOptions = @{
        FilePath = $uninstallCommand
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }
    if ($uninstallArgs -ne '') { $processOptions.ArgumentList = $uninstallArgs }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
