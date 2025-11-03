# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Add argument to install silently
    # Slack uses --silent for silent installation
    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "--silent"
        PassThru     = $true
        Wait         = $true
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