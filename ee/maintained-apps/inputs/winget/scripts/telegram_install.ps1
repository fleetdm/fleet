# Telegram Desktop ZIP installer script
# Telegram Desktop is distributed as a portable ZIP archive that needs to be extracted

$zipFilePath = "${env:INSTALLER_PATH}"

try {
    # Determine installation directory based on scope
    # For user scope (empty), install to LocalAppData\Programs
    # For machine scope, install to ProgramFiles
    $installDir = Join-Path $env:LOCALAPPDATA "Programs\Telegram Desktop"
    
    # Create a temporary directory for extraction
    $tempExtractDir = Join-Path $env:TEMP "TelegramDesktopExtract"
    if (Test-Path $tempExtractDir) {
        Remove-Item -Path $tempExtractDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tempExtractDir -Force | Out-Null
    
    # Extract ZIP file to temporary directory
    Expand-Archive -Path $zipFilePath -DestinationPath $tempExtractDir -Force
    
    # Find Telegram.exe in the extracted contents (could be in root or subdirectory)
    $telegramExe = Get-ChildItem -Path $tempExtractDir -Filter "Telegram.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    
    if (-not $telegramExe) {
        Write-Host "Error: Telegram.exe not found in ZIP archive"
        Remove-Item -Path $tempExtractDir -Recurse -Force -ErrorAction SilentlyContinue
        Exit 1
    }
    
    # Get the directory containing Telegram.exe (could be root or subdirectory)
    $sourceDir = $telegramExe.DirectoryName
    
    # Create installation directory if it doesn't exist
    if (Test-Path $installDir) {
        Remove-Item -Path $installDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    
    # Copy all files from source directory to installation directory
    Copy-Item -Path "$sourceDir\*" -Destination $installDir -Recurse -Force
    
    # Clean up temporary extraction directory
    Remove-Item -Path $tempExtractDir -Recurse -Force -ErrorAction SilentlyContinue
    
    # Verify installation succeeded
    $installedExe = Join-Path $installDir "Telegram.exe"
    if (-not (Test-Path $installedExe)) {
        Write-Host "Error: Telegram.exe not found after installation"
        Exit 1
    }
    
    # Extract version from ZIP filename (format: tportable-x64.VERSION.zip)
    $version = "0.0.0"
    if ($zipFilePath -match 'tportable-x64\.(\d+\.\d+\.\d+)\.zip') {
        $version = $matches[1]
    } elseif ($zipFilePath -match '\.(\d+\.\d+\.\d+)\.zip') {
        $version = $matches[1]
    }
    
    Write-Host "Extracted version from filename: $version"
    
    # Create registry entry so Telegram appears in Add/Remove Programs and can be detected by osquery
    # Use HKCU since this is a user-scope installation
    $registryPath = "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\TelegramDesktop"
    try {
        # Remove existing entry if it exists
        if (Test-Path $registryPath) {
            Remove-Item -Path $registryPath -Force -ErrorAction SilentlyContinue
        }
        
        # Create new registry entry
        New-Item -Path $registryPath -Force | Out-Null
        
        # Set registry values
        # Both Version and DisplayVersion are set for compatibility with different osquery versions
        Set-ItemProperty -Path $registryPath -Name "DisplayName" -Value "Telegram Desktop" -Type String
        Set-ItemProperty -Path $registryPath -Name "Publisher" -Value "Telegram FZ-LLC" -Type String
        Set-ItemProperty -Path $registryPath -Name "Version" -Value $version -Type String
        Set-ItemProperty -Path $registryPath -Name "DisplayVersion" -Value $version -Type String
        Set-ItemProperty -Path $registryPath -Name "InstallLocation" -Value $installDir -Type String
        Set-ItemProperty -Path $registryPath -Name "UninstallString" -Value "powershell.exe -Command `"& {Remove-Item -Path '$installDir' -Recurse -Force}`"" -Type String
        Set-ItemProperty -Path $registryPath -Name "NoModify" -Value 1 -Type DWord
        Set-ItemProperty -Path $registryPath -Name "NoRepair" -Value 1 -Type DWord
        
        Write-Host "Created registry entry for Telegram Desktop"
        
        # Give osquery a moment to refresh its cache
        Start-Sleep -Seconds 2
    } catch {
        Write-Host "Warning: Could not create registry entry: $_"
        # Don't fail installation if registry creation fails
    }
    
    Write-Host "Telegram Desktop installed successfully to: $installDir"
    Exit 0
    
} catch {
    Write-Host "Error: $_"
    Exit 1
}

