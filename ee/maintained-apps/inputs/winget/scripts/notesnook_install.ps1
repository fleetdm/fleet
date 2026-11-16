# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Notesnook installs per user and has no machine-wide mode, so installing it from this
# script's SYSTEM context would put it in SYSTEM's profile where nobody can launch
# it. Run the installer as the signed-in user instead.

$exeFilePath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-notesnook"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$exitCode = 0
$stageDir = $null

try {
    $owner = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' -ErrorAction SilentlyContinue |
        Invoke-CimMethod -MethodName GetOwner -ErrorAction SilentlyContinue |
        Where-Object { $_.User } |
        Select-Object -First 1
    if (-not $owner) {
        Throw "Notesnook installs per user and no user is signed in to this host. Sign in and try again."
    }
    $userAccount = "$($owner.Domain)\$($owner.User)"
    $sid = (New-Object System.Security.Principal.NTAccount($userAccount)).Translate(
        [System.Security.Principal.SecurityIdentifier]).Value
    Write-Host "Installing Notesnook for $userAccount."

    # Earlier versions of this script installed Notesnook into SYSTEM's profile. That
    # copy can't be launched, and its old version would keep the patch policy failing.
    # It's deleted directly because its uninstaller's running-app check can hang as SYSTEM.
    foreach ($root in @('SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall', 'SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall')) {
        foreach ($sub in (Get-ChildItem -Path "Registry::HKEY_USERS\S-1-5-18\$root" -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if ($key.DisplayName -notmatch '^Notesnook \d' -or $key.Publisher -ne "Streetwriters") { continue }
            if ($key.UninstallString -notmatch '^\s*"([^"]+)"') { continue }
            # The 32-bit installer wrote it under SysWOW64 but registered the system32 path.
            $legacyDir = Split-Path $matches[1] -Parent
            if (-not (Test-Path -LiteralPath $legacyDir)) { $legacyDir = $legacyDir -replace '(?i)\\system32\\', '\SysWOW64\' }
            if ($legacyDir -notmatch '(?i)^(.+)\\AppData\\Local\\Programs\\@notesnookdesktop$') { continue }
            $legacyProfile = $matches[1]
            Write-Host "Removing the copy of Notesnook in $legacyDir"
            Get-Process -ErrorAction SilentlyContinue |
                Where-Object { $_.Path -and $_.Path.StartsWith("$legacyDir\", [System.StringComparison]::OrdinalIgnoreCase) } |
                Stop-Process -Force -ErrorAction SilentlyContinue
            for ($i = 0; $i -lt 10 -and (Test-Path -LiteralPath $legacyDir); $i++) {
                if ($i) { Start-Sleep -Seconds 1 }
                Remove-Item -LiteralPath $legacyDir -Recurse -Force -ErrorAction SilentlyContinue
            }
            if (Test-Path -LiteralPath $legacyDir) {
                Write-Host "Could not remove $legacyDir; uninstalling Notesnook will retry."
                continue
            }
            # Its shortcuts, and the 100 MB copy of the installer it kept for updates.
            foreach ($leftover in @('AppData\Local\@notesnookdesktop-updater',
                    'AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Notesnook.lnk', 'Desktop\Notesnook.lnk')) {
                Remove-Item -LiteralPath (Join-Path $legacyProfile $leftover) -Recurse -Force -ErrorAction SilentlyContinue
            }
            Remove-Item -Path $sub.PSPath -Recurse -Force -ErrorAction SilentlyContinue
            Remove-Item -Path 'Registry::HKEY_USERS\S-1-5-18\SOFTWARE\a05a6719-4910-5e6c-a2aa-9af71cd1063b' -Recurse -Force -ErrorAction SilentlyContinue
        }
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
    $stageDir = Join-Path $env:ProgramData ("fleet-notesnook-" + [guid]::NewGuid().ToString("N"))
    [void][System.IO.Directory]::CreateDirectory($stageDir, $security)
    $stagedInstaller = Join-Path $stageDir (Split-Path $exeFilePath -Leaf)
    Copy-Item -LiteralPath $exeFilePath -Destination $stagedInstaller -Force -ErrorAction Stop

    $installArgs = "/S"
    if ($installArgs) {
        $action = New-ScheduledTaskAction -Execute $stagedInstaller -Argument $installArgs
    } else {
        $action = New-ScheduledTaskAction -Execute $stagedInstaller
    }
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
