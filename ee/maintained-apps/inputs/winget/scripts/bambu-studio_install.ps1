# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# A silent install closes Bambu Studio if it's running. The installer also runs
# its bundled WebView2 Runtime and Visual C++ 2019 installers, which can leave
# updater processes behind, so this script waits on the installer itself rather
# than on everything it starts.

$exeFilePath = "${env:INSTALLER_PATH}"
$timeoutMinutes = 30

try {
    $process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/S" -PassThru
    # Without a cached handle, ExitCode is empty once the process has exited.
    $null = $process.Handle

    if (-not $process.WaitForExit($timeoutMinutes * 60 * 1000)) {
        Write-Host "The installer didn't finish within $timeoutMinutes minutes."
        & taskkill.exe /PID $process.Id /T /F | Out-Null
        Exit 1
    }

    $exitCode = $process.ExitCode
    if ($null -eq $exitCode) {
        Write-Host "The installer exited without reporting an exit code."
        Exit 1
    }
    Write-Host "Install exit code: $exitCode"

    # 3010 = success, restart required; 1641 = success, restart initiated.
    if ($exitCode -in @(0, 3010, 1641)) {
        Exit 0
    }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
