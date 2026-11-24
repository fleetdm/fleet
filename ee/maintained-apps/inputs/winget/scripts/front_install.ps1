# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# This installer puts Front in the profile of whoever runs it, so running it from
# this script's SYSTEM context would leave it where nobody can launch it. Run the
# installer as the signed-in user instead.

$exeFilePath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-front"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$appDir = "AppData\Local\Programs\Front"  # in each user's profile
$exitCode = 0
$stageDir = $null

# Matching on the install directory rather than a process name leaves a copy in
# another user's profile running.
function Get-FrontProcesses {
    param([string]$Dir)

    return @(Get-Process -ErrorAction SilentlyContinue |
        Where-Object { $_.Path -and $_.Path.StartsWith("$Dir\", [System.StringComparison]::OrdinalIgnoreCase) })
}

try {
    $owner = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' -ErrorAction SilentlyContinue |
        Invoke-CimMethod -MethodName GetOwner -ErrorAction SilentlyContinue |
        Where-Object { $_.User } |
        Select-Object -First 1
    if (-not $owner) {
        Throw "Front installs per user and no user is signed in to this host. Sign in and try again."
    }
    $userAccount = "$($owner.Domain)\$($owner.User)"
    $sid = (New-Object System.Security.Principal.NTAccount($userAccount)).Translate(
        [System.Security.Principal.SecurityIdentifier]).Value
    Write-Host "Installing Front for $userAccount."

    # Fleet's installer directory is not readable by that user, so stage a copy in
    # a new folder only SYSTEM and Administrators can write to.
    $security = New-Object System.Security.AccessControl.DirectorySecurity
    $security.SetAccessRuleProtection($true, $false)
    foreach ($grant in @(@("S-1-5-18", "FullControl"), @("S-1-5-32-544", "FullControl"), @($sid, "ReadAndExecute"))) {
        $security.AddAccessRule((New-Object System.Security.AccessControl.FileSystemAccessRule(
            (New-Object System.Security.Principal.SecurityIdentifier($grant[0])), $grant[1],
            "ContainerInherit, ObjectInherit", "None", "Allow")))
    }
    $stageDir = Join-Path $env:ProgramData ("fleet-front-" + [guid]::NewGuid().ToString("N"))
    [void][System.IO.Directory]::CreateDirectory($stageDir, $security)
    $stagedInstaller = Join-Path $stageDir (Split-Path $exeFilePath -Leaf)
    Copy-Item -LiteralPath $exeFilePath -Destination $stagedInstaller -Force -ErrorAction Stop

    # A silent install aborts with exit code 2 instead of installing while Front is
    # running, so close that user's copy first.
    $profilePath = (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList\$sid" -ErrorAction SilentlyContinue).ProfileImagePath
    if ($profilePath) {
        $installDir = Join-Path $profilePath $appDir
        if (Get-FrontProcesses $installDir) {
            Write-Host "Closing Front before installing."
            Get-FrontProcesses $installDir | Stop-Process -Force -ErrorAction SilentlyContinue
            $waited = [System.Diagnostics.Stopwatch]::StartNew()
            while ((Get-FrontProcesses $installDir) -and $waited.Elapsed.TotalSeconds -lt 60) {
                Start-Sleep -Seconds 1
            }
        }
    }

    $action = New-ScheduledTaskAction -Execute $stagedInstaller -Argument "/S"
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    $principal = New-ScheduledTaskPrincipal -UserId $userAccount
    $task = New-ScheduledTask -Action $action -Settings $settings -Principal $principal
    Register-ScheduledTask -TaskName $taskName -InputObject $task -Force -ErrorAction Stop | Out-Null

    $startDate = Get-Date
    $lastRun = (Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction Stop).LastRunTime
    Start-ScheduledTask -TaskName $taskName -ErrorAction Stop

    # Wait for a result rather than for the "Running" state, which a fast task can
    # enter and leave between polls. A task that's still queued hasn't updated
    # LastRunTime yet.
    Start-Sleep -Seconds 2
    while ($true) {
        $info = Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction Stop
        $state = (Get-ScheduledTask -TaskName $taskName -ErrorAction Stop).State
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

# Exit turns a code above Int32.MaxValue, such as a task's HRESULT, into 0.
if ($exitCode -gt [int]::MaxValue) { $exitCode = 1 }
Exit $exitCode
