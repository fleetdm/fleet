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
    # Based on: https://silentinstallhq.com/sublime-text-install-and-uninstall-powershell/
    # /VERYSILENT = Very silent installation (no dialogs)
    # /NORESTART = Do not restart the computer
    # /TASKS=contextentry = Add context menu entry (optional, can be removed if not needed)
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/VERYSILENT /NORESTART /LOG=C:\Windows\Logs\Software\SublimeText-Install.log"
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

