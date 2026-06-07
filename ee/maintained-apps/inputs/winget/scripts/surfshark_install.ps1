# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Surfshark ships as an InstallShield setup.exe that wraps an MSI.
# Silent switches come from the winget manifest InstallerSwitches.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    # InstallShield silent switches:
    # /exenoui = no setup.exe UI
    # /quiet   = pass /qn to the embedded MSI
    # /norestart = do not restart
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/exenoui /quiet /norestart"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    Write-Host "Starting Surfshark install with: $($processOptions.ArgumentList)"
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # Treat MSI reboot codes as success.
    if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
        Write-Host "Install succeeded (reboot required/initiated)."
        Exit 0
    }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
