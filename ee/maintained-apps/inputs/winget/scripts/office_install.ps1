# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Microsoft Office uses the Office Deployment Tool with /configure
    # Using Microsoft's official Fleet/Winget configuration URL for silent install
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/configure https://aka.ms/fhlwingetconfig"
        PassThru = $true
        Wait = $true
    }

    # Start process and track exit code
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    # Prints the exit code
    Write-Host "Install exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
