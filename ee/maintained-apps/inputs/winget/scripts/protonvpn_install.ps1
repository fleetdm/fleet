# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Proton VPN ships as an Inno Setup installer.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /ALLUSERS"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    Write-Host "Starting Proton VPN install with: $($processOptions.ArgumentList)"
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
