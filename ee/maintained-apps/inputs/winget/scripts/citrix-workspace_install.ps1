# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Citrix Workspace app ships as a vendor bootstrapper (CitrixWorkspaceApp.exe).
# /silent /norestart is Citrix's documented unattended install. 3010 (reboot
# required) is a normal success code for this installer.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/silent /norestart"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
    if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
        Exit 0
    }

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
