# Ollama installs per user, so its uninstall entry is in the installing user's
# registry hive rather than SYSTEM's, and its uninstaller reads the directory to
# remove out of the hive of whoever runs it -- run as SYSTEM against a user's
# install it exits 0 and deletes nothing. So search every hive and run the
# uninstaller as the user who owns the entry. A copy belonging to a user who
# isn't signed in can't be removed and fails the uninstall.

$displayNamePattern = '^Ollama version '
$publisher = "Ollama"
$taskName = "fleet-uninstall-ollama"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$timeoutSeconds = 300
$exitCode = 0

function Get-AppEntries {
    param([switch]$AnyName)

    $roots = [System.Collections.Generic.List[string]]::new()
    $roots.Add('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall')
    $roots.Add('HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall')
    # Only SYSTEM's hive and user hives (S-1-5-21 local or AD, S-1-12-1 Entra ID).
    # LocalService and NetworkService can write their own hives, and an entry
    # there would run as SYSTEM.
    foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
        if ($hive.PSChildName -notmatch '^(S-1-5-18|S-1-5-21-[\d-]+|S-1-12-1-[\d-]+)$') { continue }
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
            if (-not $AnyName -and $name -notmatch $displayNamePattern) { continue }
            if (($key.Publisher -replace "`0", "").Trim() -ne $publisher) { continue }

            # Only a real user's entry needs the uninstaller run for them; HKLM and
            # SYSTEM's hive are already in the right context.
            $sid = $null
            if ($sub.PSPath -match 'HKEY_USERS\\(S-1-5-21-[\d-]+|S-1-12-1-[\d-]+)\\') { $sid = $matches[1] }

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

# Copies in each user profile's default folder. Registry hives of users who
# aren't signed in aren't loaded, so their copies can only be found this way. A
# signed-in user's copy with no registration is what an earlier uninstall
# couldn't delete.
function Get-ProfileCopies {
    foreach ($profileKey in (Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList' -ErrorAction SilentlyContinue)) {
        $sid = $profileKey.PSChildName
        if ($sid -notmatch '^S-1-(5-21|12-1)-[\d-]+$') { continue }
        $profilePath = (Get-ItemProperty $profileKey.PSPath -ErrorAction SilentlyContinue).ProfileImagePath
        if (-not $profilePath) { continue }
        $dir = Join-Path $profilePath "AppData\Local\Programs\Ollama"
        if (-not (Test-Path -LiteralPath (Join-Path $dir "ollama.exe"))) { continue }
        [PSCustomObject]@{
            Sid         = $sid
            ProfilePath = $profilePath
            Dir         = $dir
            SignedIn    = (Test-Path "Registry::HKEY_USERS\$sid")
        }
    }
}

# Anything running out of an install folder blocks its removal. Matching on the
# folder rather than a process name keeps another user's copy running.
function Stop-AppProcesses {
    param([string]$Dir)

    Get-Process -ErrorAction SilentlyContinue |
        Where-Object { $_.Path -and $_.Path.StartsWith("$Dir\", [System.StringComparison]::OrdinalIgnoreCase) } |
        Stop-Process -Force -ErrorAction SilentlyContinue
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
    }

    # A 32-bit installer run as SYSTEM is redirected into SysWOW64, but records the
    # unredirected system32 path that 64-bit PowerShell cannot resolve.
    if ($exePath -and -not (Test-Path -LiteralPath $exePath)) {
        $redirected = $exePath -replace '(?i)\\system32\\', '\SysWOW64\'
        if ($redirected -ne $exePath -and (Test-Path -LiteralPath $redirected)) {
            Write-Host "  Uninstaller is under SysWOW64, not the recorded $exePath"
            $exePath = $redirected
        }
    }

    if ($arguments -notmatch '(?i)/VERYSILENT') {
        $arguments = "$arguments /VERYSILENT /SUPPRESSMSGBOXES /NORESTART".Trim()
    }

    return [PSCustomObject]@{ ExePath = $exePath; Arguments = $arguments }
}

function Invoke-AsUser {
    param([string]$Sid, [string]$ExePath, [string]$Arguments)

    $account = (New-Object System.Security.Principal.SecurityIdentifier($Sid)).Translate(
        [System.Security.Principal.NTAccount]).Value
    Write-Host "  Running as ${account}: $ExePath $Arguments"

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

# A user can write to their own profile, including swapping a folder for a link
# elsewhere, so their folder is removed as them rather than as SYSTEM.
function Remove-AppDir {
    param([string]$Path, [string]$Sid)

    if ($Sid) {
        try {
            $null = Invoke-AsUser -Sid $Sid -ExePath "$env:SystemRoot\System32\cmd.exe" -Arguments "/d /c rd /s /q `"$Path`""
        } catch {
            Write-Host "  $_"
        }
    } else {
        Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction SilentlyContinue
    }
    if (Test-Path -LiteralPath $Path) {
        Write-Host "  Could not remove $Path."
        return $false
    }
    return $true
}

# Shortcuts the uninstaller didn't get to. Match on where they point rather than on
# a filename, which also catches the vendor subfolders some installers create. Read
# as SYSTEM, a user's shortcut resolves its target through SYSTEM's profile, so the
# working directory is checked too, and only the owning profile is searched.
function Remove-Shortcuts {
    param([string]$Dir, [string]$ProfilePath)

    if ($ProfilePath) {
        $roots = @((Join-Path $ProfilePath 'AppData\Roaming\Microsoft\Windows\Start Menu\Programs'), (Join-Path $ProfilePath 'Desktop'))
    } else {
        $roots = @('Programs', 'Desktop', 'CommonPrograms', 'CommonDesktopDirectory') | ForEach-Object { [Environment]::GetFolderPath($_) }
    }
    $shell = New-Object -ComObject WScript.Shell
    foreach ($root in $roots) {
        if (-not $root -or -not (Test-Path -LiteralPath $root)) { continue }
        foreach ($lnk in (Get-ChildItem -LiteralPath $root -Filter '*.lnk' -Recurse -ErrorAction SilentlyContinue)) {
            try { $link = $shell.CreateShortcut($lnk.FullName) } catch { continue }
            foreach ($path in @($link.TargetPath, $link.WorkingDirectory)) {
                if ($path -and ($path -eq $Dir -or $path.StartsWith("$Dir\", [System.StringComparison]::OrdinalIgnoreCase))) {
                    Remove-Item -LiteralPath $lnk.FullName -Force -ErrorAction SilentlyContinue
                    if (Test-Path -LiteralPath $lnk.FullName) { Write-Host "  Could not remove shortcut $($lnk.FullName)." }
                    break
                }
            }
        }
    }
}

try {
    $entries = @(Get-AppEntries)

    foreach ($entry in $entries) {
        Write-Host "Removing '$($entry.DisplayName)' ($($entry.KeyPath))"
        $uninstaller = Resolve-Uninstaller $entry.Command
        # Only a user's entry has a default folder to fall back on below.
        if (-not $uninstaller.ExePath -and ($entry.Command -or -not $entry.Sid)) {
            Write-Host "  Could not find an uninstaller in uninstall string '$($entry.Command)'."
            if ($exitCode -eq 0) { $exitCode = 1 }
            continue
        }
        $hasUninstaller = $uninstaller.ExePath -and (Test-Path -LiteralPath $uninstaller.ExePath)

        # A user's uninstall string comes from a hive that user can write to, so only
        # touch the Ollama folder in that user's own profile.
        $installDir = ""
        if ($uninstaller.ExePath) { try { $installDir = [System.IO.Path]::GetFullPath((Split-Path $uninstaller.ExePath -Parent)) } catch {} }
        $isAppDir = $installDir -and (Split-Path $installDir -Leaf) -eq "Ollama"
        $profilePath = $null
        if ($entry.Sid) {
            $profilePath = (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList\$($entry.Sid)" -ErrorAction SilentlyContinue).ProfileImagePath
            $userAppDir = if ($profilePath) { Join-Path $profilePath "AppData\Local\Programs\Ollama" } else { "" }
            if (-not $hasUninstaller) { $installDir = $userAppDir }
            $isAppDir = $userAppDir -and $installDir -eq $userAppDir
        }

        if ($isAppDir) { Stop-AppProcesses $installDir }

        if (-not $hasUninstaller) {
            Write-Host "  Uninstaller is missing; removing what's left of this copy."
            if ($installDir -and (Test-Path -LiteralPath $installDir) -and
                -not ($isAppDir -and (Remove-AppDir $installDir $entry.Sid))) {
                Write-Host "  Leaving this copy registered because $installDir is still there."
                if ($exitCode -eq 0) { $exitCode = 1 }
                continue
            }
            if ($isAppDir) { Remove-Shortcuts $installDir $profilePath }
            Remove-Item -Path $entry.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
            continue
        }

        # Caught here so one copy's failure doesn't skip the others.
        try {
            if ($entry.Sid) {
                $result = Invoke-AsUser -Sid $entry.Sid -ExePath $uninstaller.ExePath -Arguments $uninstaller.Arguments
            } else {
                Write-Host "  Uninstall command: $($uninstaller.ExePath) $($uninstaller.Arguments)"
                $process = Start-Process -FilePath $uninstaller.ExePath -ArgumentList $uninstaller.Arguments `
                    -NoNewWindow -PassThru -Wait -ErrorAction Stop
                $result = $process.ExitCode
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

        if ($isAppDir -and (Test-Path -LiteralPath $installDir)) {
            Write-Host "  Uninstaller left $installDir behind; removing it."
            if (-not (Remove-AppDir $installDir $entry.Sid)) {
                if ($exitCode -eq 0) { $exitCode = 1 }
                continue
            }
        }
        if ($isAppDir) { Remove-Shortcuts $installDir $profilePath }

        if (Get-ItemProperty $entry.KeyPath -ErrorAction SilentlyContinue) {
            Remove-Item -Path $entry.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    $found = $entries.Count -gt 0
    $remaining = @(Get-AppEntries)
    $signedOut = [System.Collections.Generic.List[string]]::new()
    foreach ($copy in @(Get-ProfileCopies)) {
        if (-not $copy.SignedIn) {
            try {
                $signedOut.Add((New-Object System.Security.Principal.SecurityIdentifier($copy.Sid)).Translate(
                    [System.Security.Principal.NTAccount]).Value)
            } catch {
                $signedOut.Add($copy.Sid)
            }
            continue
        }
        # A copy that's still registered is reported below.
        if (@($remaining | Where-Object { $_.Sid -eq $copy.Sid }).Count) { continue }
        # Checked again by publisher alone, so a copy registered under another name,
        # or by an install that finished after the snapshot above, keeps its folder.
        $registered = @(Get-AppEntries -AnyName | Where-Object { $_.Sid -eq $copy.Sid })
        if ($registered.Count) {
            Write-Host "Leaving $($copy.Dir) because it's registered as '$($registered[0].DisplayName)'."
            if ($exitCode -eq 0) { $exitCode = 1 }
            continue
        }

        $found = $true
        Write-Host "Removing unregistered Ollama files at $($copy.Dir)"
        Stop-AppProcesses $copy.Dir
        if (Remove-AppDir $copy.Dir $copy.Sid) {
            Remove-Shortcuts $copy.Dir $copy.ProfilePath
        } elseif ($exitCode -eq 0) {
            $exitCode = 1
        }
    }

    if ($remaining.Count -gt 0) {
        Write-Host "WARNING: $($remaining.Count) Ollama registration(s) still present:"
        $remaining | ForEach-Object { Write-Host "  - $($_.DisplayName) [$($_.KeyPath)]" }
        if ($exitCode -eq 0) { $exitCode = 1 }
    }
    if ($signedOut.Count -gt 0) {
        Write-Host "Ollama is still installed for users who aren't signed in: $($signedOut -join ', '). Run the uninstall again while they're signed in."
        if ($exitCode -eq 0) { $exitCode = 1 }
    }
    if ($exitCode -eq 0) {
        if ($found) { Write-Host "Ollama removed." } else { Write-Host "Ollama is not installed." }
    }

} catch {
    Write-Host "Error running uninstaller: $_"
    $exitCode = 1
}

# Exit turns a code above Int32.MaxValue, such as a task's HRESULT, into 0.
if ($exitCode -gt [int]::MaxValue) { $exitCode = 1 }
Exit $exitCode
