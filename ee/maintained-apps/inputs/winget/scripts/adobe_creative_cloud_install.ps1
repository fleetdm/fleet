# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Adobe Creative Cloud is installed via a small stub executable that
    # downloads and installs the full Creative Cloud Desktop app.
    # --mode=stub is the silent install switch documented in the winget manifest.
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "--mode=stub"
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Install exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
