# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Verify installer file exists
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    Write-Host "Installing Sublime Text from: $exeFilePath"
    
    # Add arguments to install silently
    # Sublime Text uses an Inno Setup-based installer
    # Based on winget manifest: https://github.com/microsoft/winget-pkgs/blob/master/manifests/s/SublimeHQ/SublimeText/4/4.0.0.420000/SublimeHQ.SublimeText.4.installer.yaml
    # /VERYSILENT = Very silent installation (no dialogs, no progress bar)
    # /SUPPRESSMSGBOXES = Suppress message boxes
    # /NORESTART = Do not restart the computer
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
    
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Write-Host "Error details: $($_.Exception.Message)"
    Exit 1
}

