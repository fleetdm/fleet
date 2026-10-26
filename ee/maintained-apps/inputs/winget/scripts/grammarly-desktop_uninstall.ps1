# Grammarly installs per user, so this script runs each signed-in user's
# uninstaller as that user. A copy belonging to a user who isn't signed in
# can't be removed and fails the uninstall. Uninstall.exe relaunches itself
# from a temp copy and exits immediately, so a copy only counts as removed
# once its uninstall entry is gone.

$displayName = "Grammarly for Windows"
$publisher = "Grammarly Inc."
$taskName = "fleet-uninstall-grammarly"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$timeoutSeconds = 600
$exitCode = 0

function Get-GrammarlyEntries {
    $roots = [System.Collections.Generic.List[string]]::new()
    $roots.Add('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall')
    $roots.Add('HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall')
    # Only SYSTEM's hive and user hives (S-1-5-21 local or AD, S-1-12-1 Entra ID).
    # LocalService and NetworkService can write their own hives, and an entry
    # there would run as SYSTEM.
    foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
        if ($hive.PSChildName -notmatch '^(S-1-5-18|S-1-5-21-[\d-]+|S-1-12-1-[\d-]+)$') { continue }
        $roots.Add("Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall")
    }

    $entries = @()
    foreach ($root in $roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if ($key.DisplayName -ne $displayName -or $key.Publisher -ne $publisher) { continue }

            # Entries from HKLM and SYSTEM's hive run as SYSTEM; the rest as their user.
            $sid = $null
            if ($sub.PSPath -match 'HKEY_USERS\\(S-1-5-21-[\d-]+|S-1-12-1-[\d-]+)\\') { $sid = $matches[1] }

            $entries += [PSCustomObject]@{
                KeyPath = $sub.PSPath
                Sid     = $sid
                Command = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
            }
        }
    }
    return $entries
}

# Registry hives of users who aren't signed in aren't loaded, so find their
# copies by the app in their profiles instead.
function Get-SignedOutInstalls {
    foreach ($profileKey in (Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList' -ErrorAction SilentlyContinue)) {
        $sid = $profileKey.PSChildName
        if ($sid -notmatch '^S-1-(5-21|12-1)-[\d-]+$' -or (Test-Path "Registry::HKEY_USERS\$sid")) { continue }
        $profilePath = (Get-ItemProperty $profileKey.PSPath -ErrorAction SilentlyContinue).ProfileImagePath
        if (-not $profilePath) { continue }
        if (-not (Test-Path -LiteralPath (Join-Path $profilePath "AppData\Local\Grammarly\DesktopIntegrations\Grammarly.Desktop.exe"))) { continue }
        try {
            (New-Object System.Security.Principal.SecurityIdentifier($sid)).Translate([System.Security.Principal.NTAccount]).Value
        } catch {
            $sid
        }
    }
}

function Invoke-UninstallerAsUser {
    param([string]$Sid, [string]$ExePath, [string]$Arguments)

    $account = (New-Object System.Security.Principal.SecurityIdentifier($Sid)).Translate(
        [System.Security.Principal.NTAccount]).Value
    Write-Host "  Running the uninstaller as $account"

    try {
        $action = New-ScheduledTaskAction -Execute $ExePath -Argument $Arguments
        $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
        $principal = New-ScheduledTaskPrincipal -UserId $account
        $task = New-ScheduledTask -Action $action -Settings $settings -Principal $principal
        Register-ScheduledTask -TaskName $taskName -InputObject $task -Force -ErrorAction Stop | Out-Null

        $startDate = Get-Date
        $lastRun = (Get-ScheduledTaskInfo -TaskName $taskName).LastRunTime
        Start-ScheduledTask -TaskName $taskName -ErrorAction Stop

        # Wait for a result rather than for the "Running" state, which a fast task can
        # enter and leave between polls. A task that's still queued hasn't updated
        # LastRunTime yet.
        Start-Sleep -Seconds 2
        while ($true) {
            $info = Get-ScheduledTaskInfo -TaskName $taskName
            $state = (Get-ScheduledTask -TaskName $taskName).State
            if ($info.LastRunTime -ne $lastRun -and $state -ne "Running" -and $info.LastTaskResult -ne $taskRunning) {
                return $info.LastTaskResult
            }
            if ((New-TimeSpan -Start $startDate).TotalSeconds -gt $timeoutSeconds) {
                Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
                Write-Host "  Timed out waiting for the uninstall task to finish."
                return 1460  # ERROR_TIMEOUT
            }
            Start-Sleep -Seconds 5
        }
    } finally {
        if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) {
            Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
        }
    }
}

try {
    # Left behind only if an install was interrupted.
    if (Get-ScheduledTask -TaskName "fleet-install-grammarly" -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName "fleet-install-grammarly" -Confirm:$false -ErrorAction SilentlyContinue
    }

    foreach ($entry in @(Get-GrammarlyEntries)) {
        Write-Host "Removing Grammarly ($($entry.KeyPath))"
        $exePath = ""
        $arguments = ""
        if ($entry.Command -match '^\s*"([^"]+)"\s*(.*)$') {
            $exePath = $matches[1]
            $arguments = $matches[2].Trim()
        } elseif ($entry.Command -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $exePath = $matches[1]
            $arguments = $matches[2].Trim()
        }
        if ($arguments -cnotmatch '(^|\s)/S($|\s)') { $arguments = "$arguments /S".Trim() }

        if (-not $exePath -or -not (Test-Path -LiteralPath $exePath)) {
            Write-Host "  Uninstaller is missing: $($entry.Command)"
            if ($exitCode -eq 0) { $exitCode = 1 }
            continue
        }

        Write-Host "  Uninstall command: $exePath $arguments"
        # Caught here so one copy's failure doesn't skip the others.
        try {
            if ($entry.Sid) {
                $result = Invoke-UninstallerAsUser -Sid $entry.Sid -ExePath $exePath -Arguments $arguments
            } else {
                $process = Start-Process -FilePath $exePath -ArgumentList $arguments -NoNewWindow -PassThru -ErrorAction Stop
                # Reading Handle now keeps ExitCode available after WaitForExit(timeout).
                $null = $process.Handle
                if ($process.WaitForExit($timeoutSeconds * 1000)) {
                    $result = $process.ExitCode
                } else {
                    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
                    Write-Host "  Timed out waiting for the uninstaller."
                    $result = 1460  # ERROR_TIMEOUT
                }
            }
        } catch {
            Write-Host "  Could not run the uninstaller: $_"
            $result = 1
        }
        Write-Host "  Uninstall exit code: $result"
        if ($result -ne 0) {
            if ($exitCode -eq 0) { $exitCode = $result }
            continue
        }

        for ($waited = 0; $waited -lt 180 -and (Test-Path $entry.KeyPath); $waited += 5) { Start-Sleep -Seconds 5 }
    }

    $remaining = @(Get-GrammarlyEntries)
    $signedOut = @(Get-SignedOutInstalls)
    if ($remaining.Count -gt 0) {
        Write-Host "WARNING: Grammarly is still registered at:"
        $remaining | ForEach-Object { Write-Host "  $($_.KeyPath)" }
        if ($exitCode -eq 0) { $exitCode = 1 }
    }
    if ($signedOut.Count -gt 0) {
        Write-Host "Grammarly is still installed for users who aren't signed in: $($signedOut -join ', '). Run the uninstall again while they're signed in."
        if ($exitCode -eq 0) { $exitCode = 1 }
    }
    if ($exitCode -eq 0) {
        Write-Host "Grammarly removed."
    }

} catch {
    Write-Host "Error running uninstaller: $_"
    $exitCode = 1
}

# Exit turns a code above Int32.MaxValue, such as a task's HRESULT, into 0.
if ($exitCode -gt [int]::MaxValue) { $exitCode = 1 }
Exit $exitCode
