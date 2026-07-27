# Learn more about .msi install scripts:
# http://fleetdm.com/learn-more-about/msi-install-scripts
#
# Mozilla VPN's MSI starts the MozillaVPNBroker / MozillaVPNProxy services
# (ServiceControl with Wait=1) and, on a fresh install, auto-launches the
# "Mozilla VPN" GUI via an async custom action. On a headless CI host those
# processes linger, and the default "Start-Process msiexec -Wait" waits on the
# whole process tree -- so the step never returns even though the product
# installs correctly, and the runner is eventually killed (~1h). Instead we
# wait on the msiexec process ITSELF (not its descendants) with a bounded
# timeout, then stop any lingering Mozilla VPN GUI process so it can't
# interfere with the rest of the validation run.

$logFile = "${env:TEMP}/fleet-install-software.log"
$msiFilePath = "${env:INSTALLER_PATH}"

$timeoutSeconds = 300
$pollIntervalSeconds = 5

# Recursively stop a process and its children (used only if msiexec wedges).
function Stop-ProcessTree {
    param([int]$ParentId)
    Get-CimInstance Win32_Process -Filter "ParentProcessId = $ParentId" -ErrorAction SilentlyContinue |
        ForEach-Object { Stop-ProcessTree -ParentId $_.ProcessId }
    Stop-Process -Id $ParentId -Force -ErrorAction SilentlyContinue
}

try {
    if (-not (Test-Path $msiFilePath)) {
        Write-Host "Error: Installer file not found at: $msiFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath     = "msiexec.exe"
        ArgumentList = "/i `"$msiFilePath`" /quiet /norestart /lv `"$logFile`""
        PassThru     = $true
    }

    # NOTE: intentionally launched WITHOUT -Wait; -Wait would block on the
    # auto-launched app / services that outlive the install.
    $process = Start-Process @processOptions
    Write-Host "Launched Mozilla VPN MSI (PID: $($process.Id))"

    $elapsed = 0
    while (-not $process.HasExited -and $elapsed -lt $timeoutSeconds) {
        Start-Sleep -Seconds $pollIntervalSeconds
        $elapsed += $pollIntervalSeconds
    }

    if (-not $process.HasExited) {
        Write-Host "msiexec did not complete within ${timeoutSeconds}s; stopping it."
        Stop-ProcessTree -ParentId $process.Id
        Get-Content $logFile -Tail 500 -ErrorAction SilentlyContinue
        Exit 1
    }

    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # Stop the auto-launched GUI so it doesn't linger into later apps in the run.
    # (This only ends the running process; it does not uninstall anything.)
    Stop-Process -Name "Mozilla VPN", "MozillaVPN" -Force -ErrorAction SilentlyContinue

    # MSI reboot-required success codes.
    if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

    if ($exitCode -ne 0) {
        Get-Content $logFile -Tail 500
    }

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
