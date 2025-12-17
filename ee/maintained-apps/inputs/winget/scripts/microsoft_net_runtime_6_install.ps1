# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Microsoft .NET Runtime uses a burn installer (WiX bootstrapper)
    # Per Winget manifest: Silent: /quiet, Custom: /norestart
    $processOptions = @{
        FilePath = $exeFilePath
        ArgumentList = "/quiet", "/norestart"
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
}

