$softwareName = "Yaak"
$softwareNameLike = "$softwareName*"
$publisherName = "Yaak"

$hkcuKey      = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey   = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($hkcuKey, $machineKey, $machineKey32) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

$foundUninstaller = $false
foreach ($key in $uninstallKeys) {
    if (-not ($key.DisplayName -eq $softwareName -or $key.DisplayName -like $softwareNameLike)) {
        continue
    }
    if (-not ($publisherName -eq "" -or $key.Publisher -like "*$publisherName*")) {
        continue
    }

    $foundUninstaller = $true

    $uninstallCommand = if ($key.QuietUninstallString) {
        $key.QuietUninstallString
    } else {
        $key.UninstallString
    }

    # Parse the uninstaller exe path from the command string
    $uninstallExe = $null
    if ($uninstallCommand -match '^"([^"]+)"') {
        $uninstallExe = $matches[1]
    } elseif ($uninstallCommand -match '^(.+?\.exe)') {
        $uninstallExe = $matches[1]
    } else {
        $uninstallExe = $uninstallCommand
    }

    # Determine the install directory
    if ($key.InstallLocation -and (Test-Path $key.InstallLocation)) {
        $installDir = $key.InstallLocation
    } else {
        $installDir = Split-Path -Parent $uninstallExe
    }

    $uninstallArgs = @("/S", "_?=$installDir")

    Write-Host "Uninstall command: $uninstallExe"
    Write-Host "Uninstall args: $uninstallArgs"

    $processOptions = @{
        FilePath = $uninstallExe
        ArgumentList = $uninstallArgs
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue

    break
}

if (-not $foundUninstaller) {
    Write-Host "Uninstall entry not found"
    Exit 0
}

} catch {
    Write-Host "Error: $_"
    Exit 1
}

Exit $exitCode
