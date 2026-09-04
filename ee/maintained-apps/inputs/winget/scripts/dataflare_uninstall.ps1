# Dataflare installs per user, so its uninstall entry is in the installing user's
# registry hive rather than SYSTEM's, and its uninstaller reads the directory to
# remove out of the hive of whoever runs it -- run as SYSTEM against a user's
# install it exits 0 and deletes nothing. So search every hive and run the
# uninstaller as the user who owns the entry.

$displayNamePattern = '^Dataflare$'
$publisher = "dataflare"
$taskName = "fleet-uninstall-dataflare"
$removedDirs = [System.Collections.Generic.List[string]]::new()
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$exitCode = 0

function Get-AppEntries {
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
            # Some installers pad these values with nulls (Fork writes "Fork" + 15 of them).
            $name = ($key.DisplayName -replace "`0", "").Trim()
            if ($name -notmatch $displayNamePattern) { continue }
            if (($key.Publisher -replace "`0", "").Trim() -ne $publisher) { continue }

            # Only a real user's entry needs the uninstaller run for them; HKLM and the
            # service SIDs are already in the right context.
            $sid = $null
            if ($sub.PSPath -match 'HKEY_USERS\\(S-1-5-21-[\d-]+)\\') { $sid = $matches[1] }

            $entries += [PSCustomObject]@{
                DisplayName = $name
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

    # A 32-bit installer run as SYSTEM is redirected into SysWOW64, but records the
    # unredirected system32 path that 64-bit PowerShell cannot resolve.
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
        if ($Arguments) {
            $action = New-ScheduledTaskAction -Execute $ExePath -Argument $Arguments
        } else {
            $action = New-ScheduledTaskAction -Execute $ExePath
        }
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
            if ((New-TimeSpan -Start $startDate).TotalSeconds -gt 300) {
                Write-Host "  Uninstall task still running after 300s; checking the result anyway."
                return $null
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
    $entries = Get-AppEntries
    if ($entries.Count -eq 0) {
        Write-Host "Dataflare is not installed."
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
        $removedDirs.Add($installDir)

        # Anything running out of this install directory blocks removal. Matching on the
        # directory rather than a process name keeps another install of the same app,
        # in another user's profile, running.
        Get-Process -ErrorAction SilentlyContinue |
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
        if ($null -ne $result -and $result -ne 0 -and $exitCode -eq 0) { $exitCode = $result }

        # An uninstaller that relaunches itself from %TEMP% exits while removal is
        # still in flight. Either the directory or the registration going away means
        # it got there.
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

    # Shortcuts sit in the installing user's profile, so an uninstall that could not
    # reach that profile leaves them behind. Match on where they point rather than on a
    # filename, which also catches the vendor subfolders some installers create.
    if ($removedDirs.Count) {
        $shell = New-Object -ComObject WScript.Shell
        foreach ($profileDir in (Get-ChildItem 'C:\Users' -Directory -ErrorAction SilentlyContinue)) {
            foreach ($root in @(
                (Join-Path $profileDir.FullName 'AppData\Roaming\Microsoft\Windows\Start Menu\Programs'),
                (Join-Path $profileDir.FullName 'Desktop')
            )) {
                if (-not (Test-Path -LiteralPath $root)) { continue }
                foreach ($lnk in (Get-ChildItem -LiteralPath $root -Filter '*.lnk' -Recurse -ErrorAction SilentlyContinue)) {
                    $target = $null
                    try { $target = $shell.CreateShortcut($lnk.FullName).TargetPath } catch {}
                    if (-not $target) { continue }
                    foreach ($dir in $removedDirs) {
                        if ($target.StartsWith($dir, [System.StringComparison]::OrdinalIgnoreCase)) {
                            Remove-Item -LiteralPath $lnk.FullName -Force -ErrorAction SilentlyContinue
                            break
                        }
                    }
                }
            }
        }
    }

    $remaining = Get-AppEntries
    if ($remaining.Count -gt 0) {
        Write-Host "WARNING: $($remaining.Count) Dataflare registration(s) still present:"
        $remaining | ForEach-Object { Write-Host "  - $($_.DisplayName) [$($_.KeyPath)]" }
        if ($exitCode -eq 0) { $exitCode = 1 }
    } else {
        Write-Host "Dataflare removed."
    }

} catch {
    Write-Host "Error running uninstaller: $_"
    $exitCode = 1
}

Exit $exitCode
