# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Verify installer file exists
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    Write-Host "Installing Telegram Desktop from: $exeFilePath"
    
    # Add arguments to install silently
    # Telegram uses an Inno Setup-based installer
    # Try /VERYSILENT first (more reliable for Inno Setup), fall back to /S if needed
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }
    
    # Start process and track exit code
    Write-Host "Starting installation with arguments: $($processOptions.ArgumentList)"
    $process = Start-Process @processOptions
    
    if ($null -eq $process) {
        Write-Host "Error: Failed to start installer process"
        Exit 1
    }
    
    $exitCode = $process.ExitCode
    
    # Prints the exit code
    Write-Host "Install exit code: $exitCode"
    
    if ($exitCode -ne 0) {
        Write-Host "Warning: Installer exited with non-zero code: $exitCode"
        # Try with /S as fallback for user-scope installs
        Write-Host "Attempting fallback with /S switch..."
        $fallbackOptions = @{
            FilePath = "$exeFilePath"
            ArgumentList = "/S"
            PassThru = $true
            Wait = $true
            NoNewWindow = $true
        }
        $fallbackProcess = Start-Process @fallbackOptions
        if ($null -ne $fallbackProcess) {
            $fallbackExitCode = $fallbackProcess.ExitCode
            Write-Host "Fallback install exit code: $fallbackExitCode"
            Exit $fallbackExitCode
        }
    }
    
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Write-Host "Error details: $($_.Exception.Message)"
    if ($_.Exception.InnerException) {
        Write-Host "Inner exception: $($_.Exception.InnerException.Message)"
    }
    Exit 1
}
