# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Sublime Merge ships as an Inno Setup installer.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    Write-Host "Installing Sublime Merge from: $exeFilePath"

    # Inno Setup silent switches:
    # /VERYSILENT = no dialogs, no progress bar
    # /SUPPRESSMSGBOXES = suppress message boxes
    # /NORESTART = do not restart the computer
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    Write-Host "Starting installation with arguments: $($processOptions.ArgumentList)"
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Write-Host "Error details: $($_.Exception.Message)"
    Exit 1
}
