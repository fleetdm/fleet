# Locates Portfolio Performance's uninstaller from the registry and runs it silently.
# Searches HKCU first, then HKLM (incl. WOW6432Node), matching on DisplayName
# and Publisher to avoid collisions with unrelated software.

$displayNameLike = "Portfolio Performance*"
$publisher = "Andreas Buchen"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$selected = $null
foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -like $displayNameLike -and ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
    }
    if ($items) { $selected = $items | Select-Object -First 1; break }
}

if (-not $selected -or -not $selected.UninstallString) {
    Write-Host "Uninstall entry not found"
    Exit 0
}

try {
Stop-Process -Name "PortfolioPerformance" -Force -ErrorAction SilentlyContinue

    $uninstallCommand = $selected.UninstallString

    # Parse the executable path out of the uninstall string.
    $exePath = ""
    if ($uninstallCommand -match '^"([^"]+)"') {
        $exePath = $matches[1]
    } elseif ($uninstallCommand -match '^(.+?\.exe)') {
        $exePath = $matches[1]
    } else {
        $exePath = $uninstallCommand
    }

    # Determine the install directory for cleanup / NSIS _?= argument.
    $installDir = $selected.InstallLocation
    if (-not $installDir -or -not (Test-Path $installDir)) {
        $installDir = Split-Path $exePath -Parent
    }

    $uninstallArgs = @("/S", "_?=$installDir")

    Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
    Write-Host "Uninstall command: $exePath"
    Write-Host "Uninstall args: $uninstallArgs"

    $processOptions = @{
        FilePath = $exePath
        ArgumentList = $uninstallArgs
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue

    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
