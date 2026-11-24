# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Docker Desktop refuses "install --user" when it is run by the SYSTEM account,
# because that would install it into the SYSTEM profile. Run the all-users
# install instead: it needs administrator rights, puts the app in
# C:\Program Files\Docker\Docker and registers it in HKLM.
#
# Users who need Windows containers or Hyper-V VM management must also be in
# the local docker-users group; the all-users install does not add them.

$exeFilePath = "${env:INSTALLER_PATH}"
$registryKey = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Docker Desktop"

try {
    # --accept-license suppresses the subscription agreement prompt.
    $process = Start-Process -FilePath "$exeFilePath" `
        -ArgumentList "install","--accept-license","--quiet" -PassThru
    # Touch the handle so ExitCode is still readable after the process exits.
    $null = $process.Handle

    # The installer process can keep running for a while after the app is
    # registered with Windows, so stop waiting as soon as the registration is
    # there rather than blocking on the process.
    $deadline = (Get-Date).AddMinutes(8)
    while ((Get-Date) -lt $deadline) {
        if (Get-ItemProperty -Path $registryKey -ErrorAction SilentlyContinue) {
            Write-Host "Docker Desktop registered in HKLM."
            Exit 0
        }
        if ($process.HasExited) {
            # Registration can land a moment after the installer exits.
            Start-Sleep -Seconds 10
            if (Get-ItemProperty -Path $registryKey -ErrorAction SilentlyContinue) {
                Write-Host "Docker Desktop registered in HKLM."
                Exit 0
            }
            Write-Host "Install exit code: $($process.ExitCode)"
            Write-Host "The installer exited without registering Docker Desktop. See C:\ProgramData\DockerDesktop\install-log-admin.txt."
            if ($process.ExitCode -eq 0) { Exit 1 }
            Exit $process.ExitCode
        }
        Start-Sleep -Seconds 10
    }

    Write-Host "Docker Desktop did not register within 8 minutes. See C:\ProgramData\DockerDesktop\install-log-admin.txt."
    Exit 1
} catch {
    Write-Host "Error: $_"
    Exit 1
}
