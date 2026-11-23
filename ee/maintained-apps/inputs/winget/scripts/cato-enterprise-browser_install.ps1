# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Cato Browser uses the Chromium installer, which is silent by default.
# --system-level installs for all users; without it the browser lands in the
# profile of the account that runs this script. --do-not-launch-chrome keeps
# the browser from opening when the install finishes.

$exeFilePath = "${env:INSTALLER_PATH}"
$timeoutMinutes = 30

try {
    $process = Start-Process -FilePath $exeFilePath `
        -ArgumentList "--system-level --do-not-launch-chrome" `
        -PassThru
    # Holding the handle keeps the exit code readable after the installer exits.
    $null = $process.Handle

    if (-not $process.WaitForExit($timeoutMinutes * 60 * 1000)) {
        Write-Host "The installer didn't finish within $timeoutMinutes minutes."
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Exit 1
    }

    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"
    if ($null -eq $exitCode) {
        Write-Host "Couldn't read the installer's exit code."
        Exit 1
    }
    if ($exitCode -ne 0) {
        Write-Host "If Cato Browser is open, close it and try again."
    }
    if ($exitCode -gt [int]::MaxValue -or $exitCode -lt 0) { $exitCode = 1 }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
