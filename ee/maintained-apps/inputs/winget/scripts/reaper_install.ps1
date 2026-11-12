# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# REAPER ships as an NSIS (Nullsoft) installer. Silent install uses /S.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    # NSIS (Nullsoft) installers require /S for silent installation. REAPER
    # installs machine-wide under Program Files by default when run elevated,
    # so no /D override is needed.
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/S"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    Write-Host "Starting REAPER install with: $($processOptions.ArgumentList)"
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # REAPER's NSIS installer returns 1223 (ERROR_CANCELLED) on a successful
    # silent (/S) install. This is documented in the winget manifest as an
    # InstallerSuccessCode (Cockos.REAPER: InstallerSuccessCodes: [1223]); the
    # install still completes (validator confirmed the app under
    # "C:\Program Files\REAPER (x64)"). Normalize it to success.
    if ($exitCode -eq 1223) {
        Write-Host "Exit code 1223 is a documented REAPER success code; treating as success."
        $exitCode = 0
    }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
