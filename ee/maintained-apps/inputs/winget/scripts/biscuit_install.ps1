# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Biscuit ships as an NSIS (electron-builder) installer.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    # NSIS installers require /S flag for silent installation
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/S"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
