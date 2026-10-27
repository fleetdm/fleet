# Airtable installs per user, so this script runs each signed-in user's
# uninstaller as that user. A copy belonging to a user who isn't signed in
# can't be removed and fails the uninstall. Settings in each user's profile
# are left in place.

$displayName = "Airtable"
$publisher = "Airtable"
$taskName = "fleet-uninstall-airtable"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$timeoutSeconds = 600
$exitCode = 0

function Get-AirtableEntries {
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
# copies by the app folders in their profiles instead.
function Get-SignedOutInstalls {
    foreach ($profileKey in (Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList' -ErrorAction SilentlyContinue)) {
        $sid = $profileKey.PSChildName
        if ($sid -notmatch '^S-1-(5-21|12-1)-[\d-]+$' -or (Test-Path "Registry::HKEY_USERS\$sid")) { continue }
        $profilePath = (Get-ItemProperty $profileKey.PSPath -ErrorAction SilentlyContinue).ProfileImagePath
        if (-not $profilePath) { continue }
        $appDirs = @(Get-ChildItem -LiteralPath (Join-Path $profilePath "AppData\Local\Airtable") -Directory -Filter "app-*" -ErrorAction SilentlyContinue)
        if ($appDirs.Count -eq 0) { continue }
        try {
            (New-Object System.Security.Principal.SecurityIdentifier($sid)).Translate([System.Security.Principal.NTAccount]).Value
        } catch {
            $sid
        }
    }
}

function Invoke-AsUser {
    param([string]$Sid, [string]$ExePath, [string]$Arguments)

    $account = (New-Object System.Security.Principal.SecurityIdentifier($Sid)).Translate(
        [System.Security.Principal.NTAccount]).Value
    Write-Host "  Running as $account"

    try {
        $action = New-ScheduledTaskAction -Execute $ExePath -Argument $Arguments
        $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
        $principal = New-ScheduledTaskPrincipal -UserId $account
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
                return $info.LastTaskResult
            }
            if ((New-TimeSpan -Start $startDate).TotalSeconds -gt $timeoutSeconds) {
                Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
                Write-Host "  Timed out waiting for the task to finish."
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

# Retry because a virus scanner or an exiting process can hold a file briefly.
# A user's folder is deleted as that user, because a subfolder they swap for a
# junction mid-delete could redirect a SYSTEM delete elsewhere.
function Remove-AppDir {
    param([string]$Path, [string]$Sid)

    if ($Sid) {
        $quotedPath = "'" + $Path.Replace("'", "''") + "'"
        $command = "for (`$i = 0; `$i -lt 30 -and (Test-Path -LiteralPath $quotedPath); `$i++) { if (`$i) { Start-Sleep -Seconds 2 }; Remove-Item -LiteralPath $quotedPath -Recurse -Force -ErrorAction SilentlyContinue }"
        try {
            $null = Invoke-AsUser -Sid $Sid -ExePath "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe" `
                -Arguments "-NoProfile -NonInteractive -WindowStyle Hidden -Command `"$command`""
        } catch {
            Write-Host "  $_"
        }
        if (-not (Test-Path -LiteralPath $Path)) { return $true }
    } else {
        for ($attempt = 0; $attempt -lt 30; $attempt++) {
            Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction SilentlyContinue
            if (-not (Test-Path -LiteralPath $Path)) { return $true }
            Start-Sleep -Seconds 2
        }
    }
    Write-Host "  Could not remove $Path."
    return $false
}

try {
    # Left behind only if an install was interrupted.
    if (Get-ScheduledTask -TaskName "fleet-install-airtable" -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName "fleet-install-airtable" -Confirm:$false -ErrorAction SilentlyContinue
    }

    foreach ($entry in @(Get-AirtableEntries)) {
        Write-Host "Removing Airtable ($($entry.KeyPath))"
        $exePath = ""
        $arguments = ""
        if ($entry.Command -match '^\s*"([^"]+)"\s*(.*)$') {
            $exePath = $matches[1]
            $arguments = $matches[2].Trim()
        } elseif ($entry.Command -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $exePath = $matches[1]
            $arguments = $matches[2].Trim()
        }
        if ($arguments -notmatch '(?i)(^|\s)(-s|--silent)($|\s)') { $arguments = "$arguments --silent".Trim() }

        # The uninstall string comes from a hive the user can write to, so only act
        # on that user's own Airtable folder.
        $installDir = ""
        if ($exePath) { try { $installDir = [System.IO.Path]::GetFullPath((Split-Path $exePath -Parent)) } catch {} }
        $isAppDir = $installDir -and (Split-Path $installDir -Leaf) -eq "Airtable"
        if ($isAppDir -and $entry.Sid) {
            $profilePath = (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList\$($entry.Sid)" -ErrorAction SilentlyContinue).ProfileImagePath
            $isAppDir = $profilePath -and $installDir -eq (Join-Path $profilePath "AppData\Local\Airtable")
        }
        if ($isAppDir) {
            Get-Process -ErrorAction SilentlyContinue |
                Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) } |
                Stop-Process -Force -ErrorAction SilentlyContinue
        }

        if (-not $exePath -or -not (Test-Path -LiteralPath $exePath)) {
            Write-Host "  Uninstaller is missing; removing what's left of this copy."
            if ($isAppDir -and (Test-Path -LiteralPath $installDir) -and -not (Remove-AppDir $installDir $entry.Sid)) {
                if ($exitCode -eq 0) { $exitCode = 1 }
                continue
            }
            Remove-Item -Path $entry.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
            continue
        }

        Write-Host "  Uninstall command: $exePath $arguments"
        # Caught here so one copy's failure doesn't skip the others.
        try {
            if ($entry.Sid) {
                $result = Invoke-AsUser -Sid $entry.Sid -ExePath $exePath -Arguments $arguments
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
            Write-Host "  Leaving this copy registered so the uninstall can be retried."
            if ($exitCode -eq 0) { $exitCode = $result }
            continue
        }

        # The uninstaller can't delete its own Update.exe, so it leaves the install
        # folder behind.
        if ($isAppDir) {
            for ($waited = 0; $waited -lt 30; $waited++) {
                $running = @(Get-Process -ErrorAction SilentlyContinue |
                    Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) })
                if ($running.Count -eq 0) { break }
                Start-Sleep -Seconds 1
            }
            if ((Test-Path -LiteralPath $installDir) -and -not (Remove-AppDir $installDir $entry.Sid)) {
                if ($exitCode -eq 0) { $exitCode = 1 }
                continue
            }
        }
        if (Get-ItemProperty $entry.KeyPath -ErrorAction SilentlyContinue) {
            Remove-Item -Path $entry.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    $remaining = @(Get-AirtableEntries)
    $signedOut = @(Get-SignedOutInstalls)
    if ($remaining.Count -gt 0) {
        Write-Host "WARNING: Airtable is still registered at:"
        $remaining | ForEach-Object { Write-Host "  $($_.KeyPath)" }
        if ($exitCode -eq 0) { $exitCode = 1 }
    }
    if ($signedOut.Count -gt 0) {
        Write-Host "Airtable is still installed for users who aren't signed in: $($signedOut -join ', '). Run the uninstall again while they're signed in."
        if ($exitCode -eq 0) { $exitCode = 1 }
    }
    if ($exitCode -eq 0) {
        Write-Host "Airtable removed."
    }

} catch {
    Write-Host "Error running uninstaller: $_"
    $exitCode = 1
}

# Exit turns a code above Int32.MaxValue, such as a task's HRESULT, into 0.
if ($exitCode -gt [int]::MaxValue) { $exitCode = 1 }
Exit $exitCode
