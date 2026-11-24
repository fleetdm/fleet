# Bandicam and BandiFix are closed first (an active recording is stopped),
# because the uninstaller's "Bandicam is in use" prompt isn't suppressed by /S.
# The NSIS uninstaller hands off to a copy of itself and returns early, so this
# script waits for Bandicam to leave Programs and Features.

$keyName = "Bandicam"
$timeoutSeconds = 300

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Find-UninstallKey {
    foreach ($p in $paths) {
        $key = Join-Path $p $keyName
        if (Test-Path -LiteralPath $key) { return $key }
    }
    return $null
}

$key = Find-UninstallKey
if (-not $key) {
    Write-Host "Bandicam is not installed."
    Exit 0
}

try {
    $uninstallCommand = "$((Get-ItemProperty -LiteralPath $key).UninstallString)"

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

    # NSIS only accepts an uppercase /S.
    if ($existingArgs -cnotmatch '(^|\s)/S($|\s)') {
        $existingArgs = ("$existingArgs /S").Trim()
    }

    $installDir = (Split-Path -Parent $exePath).TrimEnd('\')
    $mainExe = Join-Path $installDir "bdcam.exe"

    $running = @(Get-Process -Name "bdcam", "bdfix" -ErrorAction SilentlyContinue)
    if ($running.Count -gt 0) {
        Write-Host "Closing Bandicam before uninstalling."
        $running | Stop-Process -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 2
    }

    Write-Host "Uninstall command: $exePath"
    Write-Host "Uninstall args: $existingArgs"
    Start-Process -FilePath $exePath -ArgumentList $existingArgs | Out-Null

    $sw = [Diagnostics.Stopwatch]::StartNew()
    while ((Find-UninstallKey) -and $sw.Elapsed.TotalSeconds -lt $timeoutSeconds) {
        Start-Sleep -Seconds 2
    }

    $remaining = Find-UninstallKey
    if ($remaining) {
        if (Test-Path -LiteralPath $mainExe) {
            Write-Host "Bandicam is still installed after $timeoutSeconds seconds."
            Exit 1
        }
        Write-Host "Bandicam's files are gone but it is still registered; removing the registration."
        Remove-Item -LiteralPath $remaining -Recurse -Force -ErrorAction Stop
    }

    # A silent uninstall keeps files it didn't install. Like the interactive
    # uninstaller, only clear them out of the default install folders.
    $defaultDirs = @("$env:ProgramFiles\Bandicam", "${env:ProgramFiles(x86)}\Bandicam")
    if (($defaultDirs -contains $installDir) -and (Test-Path -LiteralPath $installDir)) {
        Start-Sleep -Seconds 3
        Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
    }

    Write-Host "Bandicam was uninstalled."
    Exit 0

} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
