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
$stageDir = $null

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

    # Fleet's installer directory is not readable by that user, so stage a copy in
    # a new folder only SYSTEM and Administrators can write to.
    $security = New-Object System.Security.AccessControl.DirectorySecurity
    $security.SetAccessRuleProtection($true, $false)
    foreach ($grant in @(@("S-1-5-18", "FullControl"), @("S-1-5-32-544", "FullControl"), @($sid, "ReadAndExecute"))) {
        $security.AddAccessRule((New-Object System.Security.AccessControl.FileSystemAccessRule(
            (New-Object System.Security.Principal.SecurityIdentifier($grant[0])), $grant[1],
            "ContainerInherit, ObjectInherit", "None", "Allow")))
    }
    $stageDir = Join-Path $env:ProgramData ("fleet-github-desktop-" + [guid]::NewGuid().ToString("N"))
    [void][System.IO.Directory]::CreateDirectory($stageDir, $security)
    $stagedInstaller = Join-Path $stageDir (Split-Path $exeFilePath -Leaf)
    Copy-Item -LiteralPath $exeFilePath -Destination $stagedInstaller -Force

    $action = New-ScheduledTaskAction -Execute $stagedInstaller -Argument "--silent"
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    $principal = New-ScheduledTaskPrincipal -UserId $userAccount
    $task = New-ScheduledTask -Action $action -Settings $settings -Principal $principal
    Register-ScheduledTask -TaskName $taskName -InputObject $task -Force | Out-Null

    $startDate = Get-Date
    $lastRun = (Get-ScheduledTaskInfo -TaskName $taskName).LastRunTime
    Start-ScheduledTask -TaskName $taskName

    # Wait for a result rather than for the "Running" state, which a fast task can
    # enter and leave between polls. A task that's still queued hasn't updated
    # LastRunTime yet.
    Start-Sleep -Seconds 2
    while ($true) {
        $info = Get-ScheduledTaskInfo -TaskName $taskName
        $state = (Get-ScheduledTask -TaskName $taskName).State
        if ($info.LastRunTime -ne $lastRun -and $state -ne "Running" -and $info.LastTaskResult -ne $taskRunning) {
            $exitCode = $info.LastTaskResult
            break
        }
        if ((New-TimeSpan -Start $startDate).TotalSeconds -gt 900) {
            Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
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
            $msi = Start-Process msiexec.exe -ArgumentList "/x $productCode /quiet /norestart" -PassThru
            # Reading Handle now keeps ExitCode available after WaitForExit(timeout).
            $null = $msi.Handle
            if (-not $msi.WaitForExit(300 * 1000)) {
                Stop-Process -Id $msi.Id -Force -ErrorAction SilentlyContinue
                Throw "Timed out removing GitHub Desktop Deployment Tool $productCode."
            }
            Write-Host "Deployment Tool uninstall exit code: $($msi.ExitCode)"
            if (@(0, 3010, 1641) -notcontains $msi.ExitCode) {
                Write-Host "GitHub Desktop is installed, but the Deployment Tool is still present and may install another copy at the next sign-in."
                $exitCode = $msi.ExitCode
                break
            }
        }
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    }
    if ($stageDir -and (Test-Path -LiteralPath $stageDir)) {
        Remove-Item -LiteralPath $stageDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Exit $exitCode
