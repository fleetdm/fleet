# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

$exitCode = 0

try {
    Write-Host "Starting Adobe Creative Cloud installation (EXE installer with --silent flag)..."
    
    # Adobe Creative Cloud uses --silent for silent installation
    $processOptions = @{
        FilePath = $exeFilePath
        ArgumentList = "--silent"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }
    
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    
    Write-Host "Install exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode

