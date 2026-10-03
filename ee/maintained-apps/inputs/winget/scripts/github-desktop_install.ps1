# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# GitHub Desktop installs per user and has no machine-wide mode, so this script
# runs the installer as the signed-in user. Any GitHub Desktop Deployment Tool
# (GitHub's machine-wide MSI) is removed afterwards, since it would otherwise
# install another copy for each user at their next sign-in.

$exeFilePath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-github-desktop"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$deploymentToolUpgradeCode = "{00D8E2EE-13EA-5BEB-87F0-70EFC46A7D4A}"
$exitCode = 0
$stagedInstaller = $null

try {
    $owner = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' -ErrorAction SilentlyContinue |
        Invoke-CimMethod -MethodName GetOwner -ErrorAction SilentlyContinue |
        Where-Object { $_.User } |
        Select-Object -First 1
    if (-not $owner) {
        Throw "GitHub Desktop installs per user and no user is signed in to this host. Sign in and try again."
    }
    $userAccount = "$($owner.Domain)\$($owner.User)"
    $sid = (New-Object System.Security.Principal.NTAccount($userAccount)).Translate(
        [System.Security.Principal.SecurityIdentifier]).Value
    Write-Host "Installing GitHub Desktop for $userAccount."

    # Fleet's installer directory is not readable by that user.
    $stagedInstaller = Join-Path $env:PUBLIC (Split-Path $exeFilePath -Leaf)
    Copy-Item -Path $exeFilePath -Destination $stagedInstaller -Force

    $action = New-ScheduledTaskAction -Execute $stagedInstaller -Argument "--silent"
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

    if ($exitCode -eq 0) {
        $uninstallKey = "Registry::HKEY_USERS\$sid\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\GitHubDesktop"
        for ($waited = 0; $waited -lt 60 -and -not (Test-Path $uninstallKey); $waited++) { Start-Sleep -Seconds 1 }
        if (-not (Test-Path $uninstallKey)) {
            Throw "The installer finished but GitHub Desktop is not registered for $userAccount."
        }

        $installer = New-Object -ComObject "WindowsInstaller.Installer"
        foreach ($productCode in @($installer.RelatedProducts($deploymentToolUpgradeCode))) {
            Write-Host "Removing GitHub Desktop Deployment Tool $productCode."
            $msi = Start-Process msiexec.exe -ArgumentList "/x $productCode /quiet /norestart" -PassThru -Wait
            Write-Host "Deployment Tool uninstall exit code: $($msi.ExitCode)"
        }
    }

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
