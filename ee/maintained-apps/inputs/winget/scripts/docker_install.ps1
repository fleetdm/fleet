# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Docker Desktop's installer runs for 5+ minutes (extracts files, configures
    # service, sets up WSL2 backend). The app becomes registered well before the
    # installer process exits. Kick off the install and poll for the Uninstall
    # registry entry the installer writes when the core install completes; this
    # is what osquery's programs table reads to detect Docker Desktop.
    Start-Process -FilePath "$exeFilePath" -ArgumentList "install","--accept-license","--quiet"

    $registryKey = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Docker Desktop"
    $deadline = (Get-Date).AddMinutes(4)
    while ((Get-Date) -lt $deadline) {
        if (Get-ItemProperty -Path $registryKey -ErrorAction SilentlyContinue) {
            Write-Host "Docker Desktop registered in HKLM."
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
