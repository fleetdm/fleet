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

    # NSIS (Nullsoft) installers require /S for silent installation.
    # /D sets the install directory (must be last, unquoted) to force a
    # machine-wide install under Program Files.
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
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
