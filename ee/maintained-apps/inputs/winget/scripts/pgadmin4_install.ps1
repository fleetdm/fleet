# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# pgAdmin 4 ships as an Inno Setup installer.
# /VERYSILENT = silent install, /SUPPRESSMSGBOXES, /NORESTART per Inno conventions.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Setup fails with exit code 5 when pgAdmin 4's files are in use. Fleet runs
    # as SYSTEM, outside the user's session, so it can't ask the app to close:
    # stop it first. Unsaved changes are lost.
    $running = Get-Process -ErrorAction SilentlyContinue |
        Where-Object { $_.Path -like "*\pgAdmin 4\*" }
    foreach ($proc in $running) {
        Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    }
    $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /ALLUSERS"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    Write-Host "Starting pgAdmin 4 install with: $($processOptions.ArgumentList)"
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    # A file can stay locked for a moment after the app is stopped. Setup rolls
    # back cleanly on exit code 5, so retry once.
    if ($exitCode -eq 5 -and $running) {
        Write-Host "Install exit code: 5. Retrying in 10 seconds."
        Start-Sleep -Seconds 10
        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
    }

    Write-Host "Install exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
