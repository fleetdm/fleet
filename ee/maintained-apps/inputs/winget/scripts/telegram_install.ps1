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
    
    Write-Host "Telegram Desktop installed successfully to: $installDir"
    Exit 0
    
} catch {
    Write-Host "Error: $_"
    Exit 1
}

