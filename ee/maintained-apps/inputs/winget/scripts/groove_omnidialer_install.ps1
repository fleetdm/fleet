# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Groove OmniDialer uses an electron-builder Nullsoft (NSIS) installer:
    # - /S for silent installation
    # - /currentuser for per-user installs
    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "/S /currentuser"
        PassThru     = $true
        Wait         = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Install exit code: $exitCode"
    Exit $exitCode
}
catch {
    Write-Host "Error: $_"
    Exit 1
}
