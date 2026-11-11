# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Antigravity IDE installs per user and has no machine-wide mode (setup ignores
# /ALLUSERS), so installing it from this script's SYSTEM context would put it in
# SYSTEM's profile where nobody can launch it. Run the installer as the signed-in
# user instead, then remove any copy an earlier version of this script left in
# SYSTEM's profile.

$exeFilePath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-antigravity-ide"
$appId = "{AA73B3E3-C6C8-45C8-B1DC-4AE56C751432}_is1"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$exitCode = 0
$stageDir = $null

function Remove-SystemProfileCopy {
    $key = "Registry::HKEY_USERS\S-1-5-18\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\$appId"
    $entry = Get-ItemProperty -LiteralPath $key -ErrorAction SilentlyContinue
    if (-not $entry) { return }

    $dir = "$($entry.InstallLocation)".TrimEnd('\')
    Write-Host "Removing the copy of Antigravity IDE in $dir."
    if ($dir) {
        Get-Process -ErrorAction SilentlyContinue |
            Where-Object { $_.Path -and $_.Path.StartsWith("$dir\", [System.StringComparison]::OrdinalIgnoreCase) } |
            Stop-Process -Force -ErrorAction SilentlyContinue
    }
    $uninstaller = $null
    if ($entry.UninstallString -match '^\s*"([^"]+)"') { $uninstaller = $matches[1] }
    # The 32-bit setup can be redirected into SysWOW64's copy of SYSTEM's profile
    # while recording the system32 path.
    if ($uninstaller -and -not (Test-Path -LiteralPath $uninstaller)) {
        $uninstaller = $uninstaller -replace '(?i)\\system32\\', '\SysWOW64\'
    }
    if ($uninstaller -and (Test-Path -LiteralPath $uninstaller)) {
        $process = Start-Process -FilePath $uninstaller -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART" `
            -NoNewWindow -PassThru
        $null = $process.Handle  # keeps ExitCode readable after the process exits
        if (-not $process.WaitForExit(300000)) {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
            Throw "Antigravity IDE was installed for $userAccount, but the uninstaller for the copy in $dir timed out."
        }
        Write-Host "Uninstall exit code: $($process.ExitCode)"
        for ($i = 0; $i -lt 60 -and (Test-Path -LiteralPath $key); $i++) { Start-Sleep -Seconds 1 }
    }
    if (Test-Path -LiteralPath $key) {
        Throw "Antigravity IDE was installed for $userAccount, but the copy in $dir could not be removed. Uninstall Antigravity IDE, then install it again."
    }
}

try {
    $owner = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' -ErrorAction SilentlyContinue |
        Invoke-CimMethod -MethodName GetOwner -ErrorAction SilentlyContinue |
        Where-Object { $_.User } |
        Select-Object -First 1
    if (-not $owner) {
        Throw "Antigravity IDE installs per user and no user is signed in to this host. Sign in and try again."
    }
    $userAccount = "$($owner.Domain)\$($owner.User)"
    $sid = (New-Object System.Security.Principal.NTAccount($userAccount)).Translate(
        [System.Security.Principal.SecurityIdentifier]).Value
    Write-Host "Installing Antigravity IDE for $userAccount."

    # Setup fails while Antigravity IDE is open, so stop the user's copy first. Unsaved
    # changes are lost.
    $profilePath = (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList\$sid" -ErrorAction SilentlyContinue).ProfileImagePath
    if ($profilePath) {
        $installDir = Join-Path $profilePath "AppData\Local\Programs\Antigravity IDE"
        $running = @(Get-Process -ErrorAction SilentlyContinue |
            Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) })
        foreach ($proc in $running) {
            Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
            Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
        }
        $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue
    }

    # Fleet's installer directory is not readable by that user, so stage a copy in
    # a new folder only SYSTEM and Administrators can write to.
    $security = New-Object System.Security.AccessControl.DirectorySecurity
    $security.SetAccessRuleProtection($true, $false)
    foreach ($grant in @(@("S-1-5-18", "FullControl"), @("S-1-5-32-544", "FullControl"), @($sid, "ReadAndExecute"))) {
        $security.AddAccessRule((New-Object System.Security.AccessControl.FileSystemAccessRule(
            (New-Object System.Security.Principal.SecurityIdentifier($grant[0])), $grant[1],
            "ContainerInherit, ObjectInherit", "None", "Allow")))
    }
    $stageDir = Join-Path $env:ProgramData ("fleet-antigravity-ide-" + [guid]::NewGuid().ToString("N"))
    [void][System.IO.Directory]::CreateDirectory($stageDir, $security)
    $stagedInstaller = Join-Path $stageDir (Split-Path $exeFilePath -Leaf)
    Copy-Item -LiteralPath $exeFilePath -Destination $stagedInstaller -Force -ErrorAction Stop

    # /MERGETASKS=!runcode keeps setup from launching Antigravity IDE when it finishes.
    $action = New-ScheduledTaskAction -Execute $stagedInstaller `
        -Argument "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CLOSEAPPLICATIONS /MERGETASKS=!runcode"
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

    if ($exitCode -eq 0) { Remove-SystemProfileCopy }

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
