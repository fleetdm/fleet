# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# DbVisualizer ships as an install4j installer.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "-q", "-Dinstall4j.suppressUnattendedReboot=true"
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
