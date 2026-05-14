# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Docker Desktop's installer self-elevates (winget manifest declares
    # ElevationRequirement: elevatesSelf). Start-Process -Wait is unreliable
    # against that pattern because the initial PID may detach while a handle
    # to a child stays open. Launch without -Wait and explicitly wait on the
    # installer process tree by name, per SilentInstallHQ's Docker Desktop
    # PowerShell guide.
    Start-Process -FilePath "$exeFilePath" -ArgumentList "install","--accept-license","--quiet"

    Start-Sleep -Seconds 5
    Get-Process -Name "Docker Desktop Installer" -ErrorAction SilentlyContinue | Wait-Process

    Write-Host "Docker Desktop install complete."
    Exit 0
} catch {
    Write-Host "Error: $_"
    Exit 1
}
