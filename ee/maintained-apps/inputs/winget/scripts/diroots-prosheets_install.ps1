# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# ProSheets ships as an Advanced Installer setup wrapping an MSI. The AI
# command-line syntax runs the embedded MSI silently: "/i //" launches the
# install, and MSI properties (accept_eula=1) follow the "//" delimiter.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/i // /qn /norestart accept_eula=1"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
    if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
