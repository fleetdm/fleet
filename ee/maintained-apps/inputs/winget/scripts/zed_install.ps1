# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Zed installs per user and has no machine-wide mode, so installing it from this
# script's SYSTEM context would put it in SYSTEM's profile where nobody can launch
# it. Run the installer as the signed-in user instead, then remove any copy an
# earlier version of this script left in SYSTEM's profile.

$exeFilePath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-zed"
$appId = "{2DB0DA96-CA55-49BB-AF4F-64AF36A86712}_is1"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$exitCode = 0
$stageDir = $null

function Remove-SystemProfileCopy {
    $key = "Registry::HKEY_USERS\S-1-5-18\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\$appId"
    $entry = Get-ItemProperty -LiteralPath $key -ErrorAction SilentlyContinue
    if (-not $entry) { return }

    $dir = "$($entry.InstallLocation)".TrimEnd('\')
    Write-Host "Removing the copy of Zed in $dir."
    if ($dir) {
        Get-Process -ErrorAction SilentlyContinue |
            Where-Object { $_.Path -and $_.Path.StartsWith("$dir\", [System.StringComparison]::OrdinalIgnoreCase) } |
            Stop-Process -Force -ErrorAction SilentlyContinue
    }
    # The uninstaller is in SysWOW64's copy of SYSTEM's profile, not next to Zed.exe.
    if ($entry.UninstallString -match '^\s*"([^"]+)"' -and (Test-Path -LiteralPath $matches[1])) {
        $process = Start-Process -FilePath $matches[1] -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART" `
            -NoNewWindow -PassThru -Wait
        Write-Host "Uninstall exit code: $($process.ExitCode)"
        for ($i = 0; $i -lt 60 -and (Test-Path -LiteralPath $key); $i++) { Start-Sleep -Seconds 1 }
    }
    if (Test-Path -LiteralPath $key) {
        Throw "Zed was installed for $userAccount, but the copy in $dir could not be removed. Uninstall Zed, then install it again."
    }
}

try {
    $owner = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' -ErrorAction SilentlyContinue |
        Invoke-CimMethod -MethodName GetOwner -ErrorAction SilentlyContinue |
        Where-Object { $_.User } |
        Select-Object -First 1
    if (-not $owner) {
        Throw "Zed installs per user and no user is signed in to this host. Sign in and try again."
    }
    $userAccount = "$($owner.Domain)\$($owner.User)"
    $sid = (New-Object System.Security.Principal.NTAccount($userAccount)).Translate(
        [System.Security.Principal.SecurityIdentifier]).Value
    Write-Host "Installing Zed for $userAccount."

    # Setup fails while Zed is open, so stop the user's copy first. Unsaved
    # changes are lost.
    $profilePath = (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList\$sid" -ErrorAction SilentlyContinue).ProfileImagePath
    if ($profilePath) {
        $installDir = Join-Path $profilePath "AppData\Local\Programs\Zed"
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
    $stageDir = Join-Path $env:ProgramData ("fleet-zed-" + [guid]::NewGuid().ToString("N"))
    [void][System.IO.Directory]::CreateDirectory($stageDir, $security)
    $stagedInstaller = Join-Path $stageDir (Split-Path $exeFilePath -Leaf)
    Copy-Item -LiteralPath $exeFilePath -Destination $stagedInstaller -Force -ErrorAction Stop

    $action = New-ScheduledTaskAction -Execute $stagedInstaller -Argument "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
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
