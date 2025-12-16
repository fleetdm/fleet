# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Verify installer file exists
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    Write-Host "Installing Spotify from: $exeFilePath"
    
    # Ensure Spotify AppData directory exists (installer may need this)
    $spotifyAppDataPath = "$env:APPDATA\Spotify"
    if (-not (Test-Path $spotifyAppDataPath)) {
        Write-Host "Creating Spotify AppData directory: $spotifyAppDataPath"
        New-Item -ItemType Directory -Path $spotifyAppDataPath -Force | Out-Null
    }
    
    # Add arguments to install silently
    # Spotify installer supports /silent for silent installation
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/silent"
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
    
    # If /silent fails, try /S as fallback (some Spotify installers use /S)
    if ($exitCode -ne 0) {
        Write-Host "Installation with /silent failed (exit code: $exitCode), trying /S as fallback..."
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
    Exit 1
}

