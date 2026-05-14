# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Docker Desktop 4.72+ added a per-user vs all-user install choice for
    # Windows. All-user silent installs hang on Windows Server runners
    # (Docker Desktop is not officially supported on Windows Server). Install
    # per-user with --user instead: no admin needed, target is
    # %LOCALAPPDATA%\Programs\DockerDesktop, and uninstall info is written to
    # HKCU. --accept-license suppresses the subscription agreement prompt.
    Start-Process -FilePath "$exeFilePath" -ArgumentList "install","--user","--accept-license","--quiet" | Out-Null

    # The installer process can keep running for several minutes after the
    # app is registered with Windows. Poll the HKCU uninstall key (osquery's
    # programs table reads both HKLM and HKU) to detect when the core install
    # has completed, rather than blocking on Start-Process -Wait.
    $registryKey = "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Docker Desktop"
    $deadline = (Get-Date).AddMinutes(4)
    while ((Get-Date) -lt $deadline) {
        if (Get-ItemProperty -Path $registryKey -ErrorAction SilentlyContinue) {
            Write-Host "Docker Desktop registered in HKCU."
            Exit 0
        }
        Start-Sleep -Seconds 10
    }

    Write-Host "Docker Desktop did not register within timeout."
    Exit 1
} catch {
    Write-Host "Error: $_"
    Exit 1
}
