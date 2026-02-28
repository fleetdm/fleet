# Learn more about .msi install scripts:
# http://fleetdm.com/learn-more-about/msi-install-scripts

$zipFilePath = "${env:INSTALLER_PATH}"
$tempExtractPath = Join-Path $env:TEMP "AdobeAcrobatInstall_$(Get-Random)"

try {
    # Extract the ZIP file
    Write-Host "Extracting ZIP to: $tempExtractPath"
    Expand-Archive -Path $zipFilePath -DestinationPath $tempExtractPath -Force

    # Path to the MSI within the extracted contents
    $msiPath = Join-Path $tempExtractPath "Adobe Acrobat\AcroPro.msi"

    if (-not (Test-Path $msiPath)) {
        Write-Host "Error: MSI file not found at expected path: $msiPath"
        Exit 1
    }

    Write-Host "Installing MSI: $msiPath"

    # Install the MSI silently
    $processOptions = @{
        FilePath = "msiexec.exe"
        ArgumentList = "/i `"$msiPath`" /qn /norestart"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Install exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
} finally {
    # Clean up extracted files
    if (Test-Path $tempExtractPath) {
        Write-Host "Cleaning up temporary files..."
        Remove-Item -Path $tempExtractPath -Recurse -Force -ErrorAction SilentlyContinue
    }
}
