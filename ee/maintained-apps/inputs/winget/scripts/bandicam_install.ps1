# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Bandicam uses an NSIS installer: /S installs it silently for all users.
# Bandicam is closed first (an active recording is stopped), because the
# installer's "Bandicam is in use" prompt isn't suppressed by /S.

$exeFilePath = "${env:INSTALLER_PATH}"
$timeoutMinutes = 8

try {
    $running = @(Get-Process -Name "bdcam" -ErrorAction SilentlyContinue)
    if ($running.Count -gt 0) {
        Write-Host "Closing Bandicam before installing."
        $running | Stop-Process -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 2
    }

    $process = Start-Process -FilePath $exeFilePath -ArgumentList "/S" -PassThru
    # Reading the handle keeps ExitCode available after the process exits.
    $null = $process.Handle

    if (-not $process.WaitForExit($timeoutMinutes * 60 * 1000)) {
        Write-Host "The installer didn't finish within $timeoutMinutes minutes."
        & taskkill.exe /PID $process.Id /T /F 2>&1 | Out-Null
        Exit 1
    }

    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # 3010 = success, restart required; 1641 = success, restart initiated.
    if ($exitCode -in @(0, 1641, 3010)) {
        Exit 0
    }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
