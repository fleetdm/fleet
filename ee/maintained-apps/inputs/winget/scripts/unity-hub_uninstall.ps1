$softwareName = "Unity Hub"
$publisher = "Unity Technologies Inc."

$registryPaths = @(
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

$exitCode = 0

try {
    $key = Get-ChildItem -Path $registryPaths -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath } |
        Where-Object {
            $_.DisplayName -like "${softwareName}*" -and
            ($publisher -eq "" -or $_.Publisher -like "*${publisher}*")
        } |
        Select-Object -First 1

    if (-not $key) {
        Write-Host "Uninstall entry not found"
        Exit 0
    }

    $uninstallCommand = $key.UninstallString
    Write-Host "Uninstall command: $uninstallCommand"

    # Parse the uninstaller executable path from the UninstallString
    if ($uninstallCommand -match '^"([^"]+)"') {
        $exePath = $matches[1]
    } elseif ($uninstallCommand -match '^(.+?\.exe)') {
        $exePath = $matches[1]
    } else {
        $exePath = $uninstallCommand
    }

    # Determine the install directory
    if ($key.InstallLocation -and (Test-Path $key.InstallLocation)) {
        $installDir = $key.InstallLocation
    } else {
        $installDir = Split-Path $exePath -Parent
    }

    # NSIS uninstaller: /S for silent, _?= to point at the install dir
    $uninstallArgs = @("/S", "_?=$installDir")

    $processOptions = @{
        FilePath     = $exePath
        ArgumentList = $uninstallArgs
        PassThru     = $true
        Wait         = $true
        NoNewWindow  = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    # NSIS leaves the install directory behind after a silent uninstall
    Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue

    Exit $exitCode
} catch {
    Write-Host "Error: $_"
    Exit 1
}
