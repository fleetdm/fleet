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
# 3010 = reboot required, 1641 = reboot initiated
$expectedExitCodes = @(0, 1641, 3010)

try {
    $before = Get-ItemProperty -Path $registryKey -ErrorAction SilentlyContinue
    if ($before) {
        Write-Host "Docker Desktop $($before.DisplayVersion) is already installed; installing over it."
    }

    # --accept-license suppresses the subscription agreement prompt.
    $process = Start-Process -FilePath "$exeFilePath" `
        -ArgumentList "install","--accept-license","--quiet" -PassThru
    # Touch the handle so ExitCode is still readable after the process exits.
    $null = $process.Handle

    # Wait for this install to finish. A registration that is already there
    # belongs to the previous install and says nothing about this one.
    if (-not $process.WaitForExit(8 * 60 * 1000)) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Write-Host "The installer did not finish within 8 minutes. See C:\ProgramData\DockerDesktop\install-log-admin.txt."
        Exit 1
    }

    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"
    if ($expectedExitCodes -notcontains $exitCode) { Exit $exitCode }

    # Child processes finish the registration, so it can land a little after the
    # installer exits. Wait for the entry osquery's programs table reads.
    $deadline = (Get-Date).AddMinutes(2)
    while ((Get-Date) -lt $deadline) {
        $after = Get-ItemProperty -Path $registryKey -ErrorAction SilentlyContinue
        if ($after) {
            Write-Host "Docker Desktop $($after.DisplayVersion) registered in HKLM."
            Exit 0
        }
        Start-Sleep -Seconds 5
    }

    Write-Host "The installer exited $exitCode but Docker Desktop is not registered. See C:\ProgramData\DockerDesktop\install-log-admin.txt."
    Exit 1
} catch {
    Write-Host "Error: $_"
    Exit 1
}
