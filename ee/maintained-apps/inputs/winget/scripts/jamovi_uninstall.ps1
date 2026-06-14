# Locates the NSIS (electron-builder) uninstaller from the registry and runs it silently.

$displayName = "Jamovi Desktop"
$publisher = "The jamovi Project"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

try {
    $app = $null
    foreach ($p in $paths) {
        $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
            ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and
            ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
        }
        if ($items) { $app = $items | Select-Object -First 1; break }
    }

    if (-not $app -or -not $app.UninstallString) {
        Write-Host "Uninstall entry not found"
        Exit 0
    }

    $uninstallString = $app.UninstallString
    $exePath = $null
    if ($uninstallString -match '^"([^"]+)"') {
        $exePath = $matches[1]
    } elseif ($uninstallString -match '^(.+?\.exe)') {
        $exePath = $matches[1]
    }

    if (-not $exePath -or -not (Test-Path $exePath)) {
        Write-Host "Error: Could not locate uninstaller executable"
        Exit 1
    }

    $installDir = $app.InstallLocation
    if (-not $installDir -or -not (Test-Path $installDir)) {
        $installDir = Split-Path -Parent $exePath
    }

    $argumentList = @("/S", "_?=$installDir")

    $processOptions = @{
        FilePath = $exePath
        ArgumentList = $argumentList
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    if ($installDir -and (Test-Path $installDir)) {
        Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue
    }

    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
