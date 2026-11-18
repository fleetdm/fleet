# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Setup fails with exit code 5 when Sublime Text's files are in use. Fleet
    # runs as SYSTEM, outside the user's session, so it can't ask the app to
    # close: stop it first. Unsaved changes are lost.
    $running = Get-Process -ErrorAction SilentlyContinue |
        Where-Object { $_.Path -like "*\Sublime Text\*" }
    foreach ($proc in $running) {
        Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    }
    $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

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

    # A file can stay locked for a moment after the app is stopped. Setup rolls
    # back cleanly on exit code 5, so retry once.
    if ($exitCode -eq 5 -and $running) {
        Write-Host "Install exit code: 5. Retrying in 10 seconds."
        Start-Sleep -Seconds 10
        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
    }
    
    # Prints the exit code
    Write-Host "Install exit code: $exitCode"
    
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Write-Host "Error details: $($_.Exception.Message)"
    Exit 1
}

