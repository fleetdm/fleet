# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# balenaEtcher installs per user and has no machine-wide mode, so installing it from this
# script's SYSTEM context would put it in SYSTEM's profile where nobody can launch
# it. Run the installer as the signed-in user instead.

$exeFilePath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-balenaetcher"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$exitCode = 0
$stagedInstaller = $null

try {
    $owner = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' -ErrorAction SilentlyContinue |
        Invoke-CimMethod -MethodName GetOwner -ErrorAction SilentlyContinue |
        Where-Object { $_.User } |
        Select-Object -First 1
    if (-not $owner) {
        Throw "balenaEtcher installs per user and no user is signed in to this host. Sign in and try again."
    }
    $userAccount = "$($owner.Domain)\$($owner.User)"
    Write-Host "Installing balenaEtcher for $userAccount."

    # Fleet's installer directory is not readable by that user.
    $stagedInstaller = Join-Path $env:PUBLIC (Split-Path $exeFilePath -Leaf)
    Copy-Item -Path $exeFilePath -Destination $stagedInstaller -Force

    $installArgs = "--silent"
    if ($installArgs) {
        $action = New-ScheduledTaskAction -Execute $stagedInstaller -Argument $installArgs
    } else {
        $action = New-ScheduledTaskAction -Execute $stagedInstaller
    }
    $trigger = New-ScheduledTaskTrigger -AtLogOn
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    $principal = New-ScheduledTaskPrincipal -UserId $userAccount
    $task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings -Principal $principal
    Register-ScheduledTask -TaskName $taskName -InputObject $task -Force | Out-Null

    $startDate = Get-Date
    Start-ScheduledTask -TaskName $taskName

    # Wait for a result rather than for the "Running" state, which a fast task can
    # enter and leave between polls.
    Start-Sleep -Seconds 2
    while ($true) {
        $info = Get-ScheduledTaskInfo -TaskName $taskName
        $state = (Get-ScheduledTask -TaskName $taskName).State
        if ($state -ne "Running" -and $info.LastTaskResult -ne $taskRunning) {
            $exitCode = $info.LastTaskResult
            break
        }
        if ((New-TimeSpan -Start $startDate).TotalSeconds -gt 900) {
            Throw "Timed out waiting for the install task to finish."
        }
        Start-Sleep -Seconds 5
    }
    Write-Host "Install exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    }
    if ($stagedInstaller -and (Test-Path -LiteralPath $stagedInstaller)) {
        Remove-Item -LiteralPath $stagedInstaller -Force -ErrorAction SilentlyContinue
    }
}

Exit $exitCode
