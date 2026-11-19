# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# This application ships as an NSIS installer. Without /MULTIUSER, a silent
# install targets the current user's profile, which under Fleet's SYSTEM
# context is the SYSTEM profile, and fails there with exit code 2.
# /MULTIUSER installs MEGAsync for all users into C:\ProgramData\MEGAsync and
# registers it under HKLM.

$exeFilePath = "${env:INSTALLER_PATH}"

# 3010 = installed, reboot required to finish removing the previous version
$expectedExitCodes = @(0, 3010)

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/S /MULTIUSER"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    if ($expectedExitCodes -contains $exitCode) {
        Exit 0
    }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
