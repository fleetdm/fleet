# GitHub Desktop installs per user, so its uninstall entry is in each installing
# user's registry hive and its uninstaller must run as that user to remove their
# shortcuts and registration. This script removes every user's copy, then any
# GitHub Desktop Deployment Tool (GitHub's machine-wide MSI). Settings and
# repositories in each user's profile are left in place.

$displayName = "GitHub Desktop"
$publisher = "GitHub, Inc."
$deploymentToolUpgradeCode = "{00D8E2EE-13EA-5BEB-87F0-70EFC46A7D4A}"
$taskName = "fleet-uninstall-github-desktop"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$exitCode = 0

function Get-GitHubDesktopEntries {
    $roots = [System.Collections.Generic.List[string]]::new()
    $roots.Add('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall')
    $roots.Add('HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall')
    foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
        if ($hive.Name -match '_Classes$') { continue }
        $roots.Add("Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall")
    }

    $entries = @()
    foreach ($root in $roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if ($key.DisplayName -ne $displayName -or $key.Publisher -ne $publisher) { continue }

            $sid = $null
            if ($sub.PSPath -match 'HKEY_USERS\\(S-1-5-21-[\d-]+)\\') { $sid = $matches[1] }

            $entries += [PSCustomObject]@{
                KeyPath = $sub.PSPath
                Sid     = $sid
                Command = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
            }
        }
    }
    return $entries
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
    foreach ($entry in @(Get-GitHubDesktopEntries)) {
        Write-Host "Removing GitHub Desktop ($($entry.KeyPath))"
        $exePath = ""
        $arguments = ""
        if ($entry.Command -match '^\s*"([^"]+)"\s*(.*)$') {
            $exePath = $matches[1]
            $arguments = $matches[2].Trim()
        } elseif ($entry.Command -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $exePath = $matches[1]
            $arguments = $matches[2].Trim()
        }
        if ($arguments -notmatch '(?i)(^|\s)(-s|--silent)($|\s)') { $arguments = "$arguments -s".Trim() }

        if (-not $exePath -or -not (Test-Path -LiteralPath $exePath)) {
            Write-Host "  Uninstaller is missing; removing the leftover registration."
            Remove-Item -Path $entry.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
            continue
        }

        $installDir = Split-Path $exePath -Parent
        Get-Process -ErrorAction SilentlyContinue |
            Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) } |
            Stop-Process -Force -ErrorAction SilentlyContinue

        Write-Host "  Uninstall command: $exePath $arguments"
        if ($entry.Sid) {
            $result = Invoke-UninstallerAsUser -Sid $entry.Sid -ExePath $exePath -Arguments $arguments
        } else {
            $result = (Start-Process -FilePath $exePath -ArgumentList $arguments -NoNewWindow -PassThru -Wait).ExitCode
        }
        Write-Host "  Uninstall exit code: $result"
        if ($result -ne 0 -and $exitCode -eq 0) { $exitCode = $result }

        # The uninstaller can't delete its own Update.exe, so it leaves the install
        # folder behind.
        for ($waited = 0; $waited -lt 30; $waited++) {
            $running = @(Get-Process -ErrorAction SilentlyContinue |
                Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) })
            if ($running.Count -eq 0) { break }
            Start-Sleep -Seconds 1
        }
        if ((Split-Path $installDir -Leaf) -eq "GitHubDesktop") {
            Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
        }
        if (Get-ItemProperty $entry.KeyPath -ErrorAction SilentlyContinue) {
            Remove-Item -Path $entry.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    $installer = New-Object -ComObject "WindowsInstaller.Installer"
    foreach ($productCode in @($installer.RelatedProducts($deploymentToolUpgradeCode))) {
        Write-Host "Removing GitHub Desktop Deployment Tool $productCode"
        $msi = Start-Process msiexec.exe -ArgumentList "/x $productCode /quiet /norestart" -PassThru -Wait
        Write-Host "  Uninstall exit code: $($msi.ExitCode)"
        if (@(0, 3010, 1641) -notcontains $msi.ExitCode -and $exitCode -eq 0) { $exitCode = $msi.ExitCode }
    }

    $remaining = @(Get-GitHubDesktopEntries)
    if ($remaining.Count -gt 0) {
        Write-Host "WARNING: GitHub Desktop is still registered at:"
        $remaining | ForEach-Object { Write-Host "  $($_.KeyPath)" }
        if ($exitCode -eq 0) { $exitCode = 1 }
    } else {
        Write-Host "GitHub Desktop removed."
    }

} catch {
    Write-Host "Error running uninstaller: $_"
    $exitCode = 1
}

Exit $exitCode
