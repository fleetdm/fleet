# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Splashtop Streamer ships as an InstallShield setup.exe that extracts an
# embedded MSI to a temp dir and launches it (plus a PreVerCheck.exe helper)
# as CHILD processes. The bootstrapper's main process can return BEFORE those
# children finish, so a plain `Start-Process -Wait` declares the install done
# while msiexec is still running. That leftover msiexec holds the global
# _MSIExecute mutex (causing 1618 / "Failed to grab execution mutex" for the
# next app the validator runs) and the resident installer keeps a lock on the
# downloaded setup.exe ("Access is denied" when cleaning up).
#
# To install TRULY synchronously: run the documented silent switches, then
# block until every spawned PreVerCheck/msiexec/Splashtop installer child has
# exited before returning. Switches per Splashtop's deployment docs and the
# winget manifest:
#   prevercheck /s /i hidewindow=1,confirm_d=0
# (prevercheck + /i required; /s = silent; commas separate options, no spaces)

$exeFilePath = "${env:INSTALLER_PATH}"

# Wait (up to $TimeoutSeconds) for any process matching the given name patterns
# to exit. Used to block on the installer children the bootstrapper leaves behind.
function Wait-ForInstallerChildren {
    param(
        [string[]]$NamePatterns,
        [int]$TimeoutSeconds = 600,
        [int]$PollSeconds = 3
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        $running = @(Get-Process -ErrorAction SilentlyContinue |
            Where-Object {
                foreach ($pat in $NamePatterns) { if ($_.Name -like $pat) { return $true } }
                return $false
            })
        if ($running.Count -eq 0) { return $true }
        Write-Host "Waiting for installer child processes to exit: $($running.Name -join ', ')"
        Start-Sleep -Seconds $PollSeconds
    }
    Write-Host "Warning: installer child processes still running after ${TimeoutSeconds}s timeout."
    return $false
}

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "prevercheck /s /i hidewindow=1,confirm_d=0"
        PassThru     = $true
        Wait         = $true
        NoNewWindow  = $true
    }

    Write-Host "Starting Splashtop Streamer install with: $($processOptions.ArgumentList)"
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Bootstrapper exit code: $exitCode"

    # Block until the spawned installer children release the install mutex and
    # the setup.exe file lock, so the validator can clean up and the next app
    # can grab the mutex.
    Wait-ForInstallerChildren -NamePatterns @('PreVerCheck', 'msiexec', 'Splashtop*INSTALLER*', '*_INSTALLER*') -TimeoutSeconds 600 | Out-Null

    if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
        Write-Host "Install succeeded (reboot required/initiated)."
        Exit 0
    }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
