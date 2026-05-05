# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Adobe Acrobat Reader uses specific silent install arguments
    # -sfx_nu: No user interface for extraction
    # /sAll: Silent installation for all components
    # /rs: Suppress reboot
    # /msi: Use MSI installer contained within the EXE
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "-sfx_nu /sAll /rs /msi"
        PassThru = $true
        Wait = $true
    }

    # Start process and track exit code
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    # Prints the exit code
    Write-Host "Install exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
