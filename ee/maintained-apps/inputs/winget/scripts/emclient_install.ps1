# eM Client installs as a small "eM Client (Updater)" MSI that downloads and
# installs the eM Client app for the user who runs it, which fails as SYSTEM.
# This script installs the MSI without that step, then installs the app for
# each signed-in user. Other users get it the next time they sign in.

$installerPath = "${env:INSTALLER_PATH}"
$launcherPath = Join-Path ${env:ProgramFiles(x86)} 'eM Client\MailClient.exe'
$appInstallerUrl = 'https://licensing.emclient.com/api/update/emclient.appinstaller'
$packageName = 'eMClient.20054CA46072C'
$taskName = 'fleet-install-emclient'
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
# Fleet stops install scripts after an hour.
$deadline = (Get-Date).AddMinutes(50)
# 3010 = reboot required, 1641 = reboot initiated
$successCodes = @(0, 3010, 1641)
$workDir = Join-Path $env:TEMP "fleet-emclient-$([guid]::NewGuid())"
$logFile = Join-Path $workDir 'install.log'
$exitCode = 0

function New-SystemInstallTransform($msiPath, $transformPath) {
    $copyPath = "$transformPath.msi"
    Copy-Item -LiteralPath $msiPath -Destination $copyPath -Force -ErrorAction Stop
    $msi = New-Object -ComObject 'WindowsInstaller.Installer'
    $original = $msi.OpenDatabase($msiPath, 0)
    $modified = $msi.OpenDatabase($copyPath, 1)
    $view = $modified.OpenView("DELETE FROM InstallExecuteSequence WHERE Action = 'InstallMsixFromAppInstaller'")
    $view.Execute()
    $view.Close()
    $modified.Commit()
    $null = $modified.GenerateTransform($original, $transformPath)
    $null = $modified.CreateTransformSummaryInfo($original, $transformPath, 0, 0)
    foreach ($obj in @($view, $modified, $original, $msi)) {
        $null = [System.Runtime.InteropServices.Marshal]::ReleaseComObject($obj)
    }
}

function Install-ForUser($userAccount) {
    if ((Get-Date) -gt $deadline) {
        Throw "Ran out of time to install eM Client for $userAccount."
    }
    Write-Host "Installing eM Client for $userAccount."
    $action = New-ScheduledTaskAction -Execute $launcherPath -Argument "--install-msix `"$appInstallerUrl`""
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Minutes 20)
    $principal = New-ScheduledTaskPrincipal -UserId $userAccount
    $task = New-ScheduledTask -Action $action -Settings $settings -Principal $principal
    Register-ScheduledTask -TaskName $taskName -InputObject $task -Force | Out-Null

    $taskDeadline = (Get-Date).AddMinutes(20)
    if ($taskDeadline -gt $deadline) { $taskDeadline = $deadline }
    $lastRun = (Get-ScheduledTaskInfo -TaskName $taskName).LastRunTime
    Start-ScheduledTask -TaskName $taskName
    # Wait for a result rather than for the "Running" state, which a fast task can
    # enter and leave between polls. A task that's still queued hasn't updated
    # LastRunTime yet.
    Start-Sleep -Seconds 2
    while ($true) {
        $info = Get-ScheduledTaskInfo -TaskName $taskName
        $state = (Get-ScheduledTask -TaskName $taskName).State
        if ($info.LastRunTime -ne $lastRun -and $state -ne 'Running' -and $info.LastTaskResult -ne $taskRunning) {
            break
        }
        if ((Get-Date) -gt $taskDeadline) {
            Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
            Throw "Timed out installing eM Client for $userAccount."
        }
        Start-Sleep -Seconds 5
    }
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false

    if ($info.LastTaskResult -ne 0) {
        Throw "eM Client didn't install for $userAccount (exit code $($info.LastTaskResult))."
    }
    $package = Get-AppxPackage -User $userAccount -Name $packageName -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $package) {
        Throw "eM Client didn't install for $userAccount."
    }
    Write-Host "Installed eM Client $($package.Version) for $userAccount."
}

try {
    New-Item -ItemType Directory -Path $workDir -Force | Out-Null
    $transformPath = Join-Path $workDir 'system-install.mst'
    New-SystemInstallTransform $installerPath $transformPath

    $msiArgs = "/i `"$installerPath`" TRANSFORMS=`"$transformPath`" /quiet /norestart /lv `"$logFile`""
    $process = Start-Process msiexec.exe -ArgumentList $msiArgs -Wait -PassThru
    Write-Host "MSI install exited $($process.ExitCode)"
    if ($successCodes -notcontains $process.ExitCode) {
        Get-Content -LiteralPath $logFile -Tail 100 -ErrorAction SilentlyContinue
        $exitCode = $process.ExitCode
    } else {
        $users = @(Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' -ErrorAction SilentlyContinue |
            Invoke-CimMethod -MethodName GetOwner -ErrorAction SilentlyContinue |
            Where-Object { $_.User } |
            ForEach-Object { "$($_.Domain)\$($_.User)" } |
            Sort-Object -Unique)
        if ($users.Count -eq 0) {
            Write-Host "No user is signed in. eM Client will be installed for each user when they sign in."
        }
        # Uninstalling eM Client for all users leaves it marked for removal at each
        # user's next sign-in. Clear those marks so this install isn't undone.
        Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Appx\AppxAllUserStore\EndOfLife' -ErrorAction SilentlyContinue |
            Get-ChildItem -ErrorAction SilentlyContinue |
            Where-Object { $_.PSChildName -like "${packageName}_*" } |
            Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
        foreach ($userAccount in $users) {
            try {
                Install-ForUser $userAccount
            } catch {
                Write-Host "Error: $_"
                $exitCode = 1
            }
        }
    }
} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) {
        Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    }
    Remove-Item -LiteralPath $workDir -Recurse -Force -ErrorAction SilentlyContinue
}

Exit $exitCode
