# Learn more about .exe uninstall scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$softwareName = "Reqable"
$softwareNameLike = "*$softwareName*"
$publisher = ""

$userKey = `
 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$userKey32on64 = `
 'HKCU:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey = `
 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = `
 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($userKey, $userKey32on64, $machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

$foundUninstaller = $false
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -like $softwareNameLike -and
        ($publisher -eq "" -or $key.Publisher -like "*$publisher*")) {
        $foundUninstaller = $true

        $uninstallCommand = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        # Parse the uninstaller executable path from the uninstall command.
        $uninstallExe = ""
        if ($uninstallCommand -match '^"([^"]+)"') {
            $uninstallExe = $matches[1]
        } elseif ($uninstallCommand -match '^(.+?\.exe)') {
            $uninstallExe = $matches[1]
        }

        if ($uninstallExe -eq "") {
            Write-Host "Could not parse uninstaller path from: $uninstallCommand"
            $exitCode = 1
            break
        }

        # Determine the install directory.
        $installDir = ""
        if ($key.InstallLocation -and (Test-Path $key.InstallLocation)) {
            $installDir = $key.InstallLocation
        } else {
            $installDir = Split-Path -Parent $uninstallExe
        }

        $uninstallArgs = @("/VERYSILENT", "/NORESTART")

        Write-Host "Uninstall command: $uninstallExe"
        Write-Host "Uninstall args: $uninstallArgs"

        $processOptions = @{
            FilePath = $uninstallExe
            ArgumentList = $uninstallArgs
            PassThru = $true
            Wait = $true
            NoNewWindow = $true
        }

        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
        Write-Host "Uninstall exit code: $exitCode"

        if ($installDir -and (Test-Path $installDir)) {
            Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue
        }
        break
    }
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
