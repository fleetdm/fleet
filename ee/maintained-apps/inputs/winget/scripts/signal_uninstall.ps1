# Signal installs per user, so its uninstall entry is in the installing user's
# registry hive rather than SYSTEM's, and its uninstaller reads the directory to
# remove out of the hive of whoever runs it -- run as SYSTEM against a user's
# install it exits 0 and deletes nothing. So search every hive and run the
# uninstaller as the user who owns the entry.
#
# Installs left behind by older versions of this app's install script also need the
# SysWOW64 retry below: that script ran the 32-bit installer as SYSTEM, where the
# payload landed under C:\Windows\SysWOW64\config\systemprofile\... but the recorded
# UninstallString still names C:\Windows\system32\..., which 64-bit PowerShell
# cannot resolve.

$displayNameLike = "Signal*"
$publisher = "Signal Messenger, LLC"
$taskName = "fleet-uninstall-signal"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$exitCode = 0

function Get-SignalEntries {
    $roots = [System.Collections.Generic.List[string]]::new()
    $roots.Add('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall')
    $roots.Add('HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall')
    foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
        if ($hive.Name -match '_Classes$') { continue }
        $roots.Add("Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall")
        $roots.Add("Registry::$($hive.Name)\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall")
    }

    $entries = @()
    foreach ($root in $roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if (-not $key.DisplayName) { continue }
            if ($key.DisplayName -notlike $displayNameLike) { continue }
            if ($key.Publisher -notlike "$publisher*") { continue }

            # Only a real user's entry needs the uninstaller run for them; HKLM and the
            # service SIDs are already in the right context.
            $sid = $null
            if ($sub.PSPath -match 'HKEY_USERS\\(S-1-5-21-[\d-]+)\\') { $sid = $matches[1] }

            $entries += [PSCustomObject]@{
                DisplayName = $key.DisplayName
                KeyPath     = $sub.PSPath
                Sid         = $sid
                Command     = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
            }
        }
    }
    return $entries
}

function Resolve-Uninstaller {
    param([string]$Command)

    $exePath = ""
    $arguments = ""
    if ($Command -match '^\s*"([^"]+)"\s*(.*)$') {
        $exePath = $matches[1]
        $arguments = $matches[2].Trim()
    } elseif ($Command -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $exePath = $matches[1]
        $arguments = $matches[2].Trim()
    } else {
        Throw "Could not parse uninstall string: $Command"
    }

    if (-not (Test-Path -LiteralPath $exePath)) {
        $redirected = $exePath -replace '(?i)\\system32\\', '\SysWOW64\'
        if ($redirected -ne $exePath -and (Test-Path -LiteralPath $redirected)) {
            Write-Host "  Uninstaller is under SysWOW64, not the recorded $exePath"
            $exePath = $redirected
        }
    }

    # \b does not match between a space and a slash, so anchor on whitespace.
    if ($arguments -notmatch '(?i)(^|\s)/S($|\s)') { $arguments = "$arguments /S".Trim() }
    return [PSCustomObject]@{ ExePath = $exePath; Arguments = $arguments }
}

function Invoke-UninstallerAsUser {
    param([string]$Sid, [string]$ExePath, [string]$Arguments)

    $account = (New-Object System.Security.Principal.SecurityIdentifier($Sid)).Translate(
        [System.Security.Principal.NTAccount]).Value
    Write-Host "  Running the uninstaller as $account"

    try {
        $action = New-ScheduledTaskAction -Execute $ExePath -Argument $Arguments
        $trigger = New-ScheduledTaskTrigger -AtLogOn
        $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
        $principal = New-ScheduledTaskPrincipal -UserId $account
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
                return $info.LastTaskResult
            }
            if ((New-TimeSpan -Start $startDate).TotalSeconds -gt 600) {
                Throw "Timed out waiting for the uninstall task to finish."
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
    $entries = Get-SignalEntries
    if ($entries.Count -eq 0) {
        Write-Host "Signal is not installed."
        Exit 0
    }

    foreach ($entry in $entries) {
        Write-Host "Removing '$($entry.DisplayName)' ($($entry.KeyPath))"
        if (-not $entry.Command) {
            Write-Host "  No uninstall string recorded; removing the leftover registration."
            Remove-Item -Path $entry.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
            continue
        }

        $uninstaller = Resolve-Uninstaller $entry.Command
        $installDir = Split-Path $uninstaller.ExePath -Parent

        # Scope the kill to this install so another user's session is left alone.
        Get-Process -Name "Signal" -ErrorAction SilentlyContinue |
            Where-Object { $_.Path -and $_.Path.StartsWith($installDir, [System.StringComparison]::OrdinalIgnoreCase) } |
            Stop-Process -Force -ErrorAction SilentlyContinue

        if (-not (Test-Path -LiteralPath $uninstaller.ExePath)) {
            Write-Host "  Uninstaller is gone from $($uninstaller.ExePath); removing the leftover registration."
            Remove-Item -Path $entry.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
            continue
        }

        Write-Host "  Uninstall command: $($uninstaller.ExePath)"
        Write-Host "  Uninstall args: $($uninstaller.Arguments)"
        if ($entry.Sid) {
            $result = Invoke-UninstallerAsUser -Sid $entry.Sid -ExePath $uninstaller.ExePath -Arguments $uninstaller.Arguments
        } else {
            $process = Start-Process -FilePath $uninstaller.ExePath -ArgumentList $uninstaller.Arguments `
                -NoNewWindow -PassThru -Wait
            $result = $process.ExitCode
        }
        Write-Host "  Uninstall exit code: $result"
        if ($result -ne 0 -and $exitCode -eq 0) { $exitCode = $result }

        # An NSIS uninstaller relaunches itself from %TEMP%, so the process we waited on
        # exits while removal is still in flight. Either the directory or the
        # registration going away means it got there.
        for ($waited = 0; $waited -lt 60; $waited++) {
            if (-not (Test-Path -LiteralPath $installDir)) { break }
            if (-not (Get-ItemProperty $entry.KeyPath -ErrorAction SilentlyContinue)) {
                Start-Sleep -Seconds 3
                break
            }
            Start-Sleep -Seconds 1
        }

        if (Test-Path -LiteralPath $installDir) {
            Write-Host "  Uninstaller left $installDir behind; removing it."
            Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
            if (Test-Path -LiteralPath $installDir) {
                Write-Host "  WARNING: could not remove $installDir."
                if ($exitCode -eq 0) { $exitCode = 1 }
            }
        }

        if (Get-ItemProperty $entry.KeyPath -ErrorAction SilentlyContinue) {
            Remove-Item -Path $entry.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    foreach ($profileDir in (Get-ChildItem 'C:\Users' -Directory -ErrorAction SilentlyContinue)) {
        foreach ($shortcut in @(
            (Join-Path $profileDir.FullName 'AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Signal.lnk'),
            (Join-Path $profileDir.FullName 'Desktop\Signal.lnk')
        )) {
            if (Test-Path -LiteralPath $shortcut) {
                Remove-Item -LiteralPath $shortcut -Force -ErrorAction SilentlyContinue
            }
        }
    }

    $remaining = Get-SignalEntries
    if ($remaining.Count -gt 0) {
        Write-Host "WARNING: $($remaining.Count) Signal registration(s) still present:"
        $remaining | ForEach-Object { Write-Host "  - $($_.DisplayName) [$($_.KeyPath)]" }
        if ($exitCode -eq 0) { $exitCode = 1 }
    } else {
        Write-Host "Signal removed."
    }

} catch {
    Write-Host "Error running uninstaller: $_"
    $exitCode = 1
}

Exit $exitCode
