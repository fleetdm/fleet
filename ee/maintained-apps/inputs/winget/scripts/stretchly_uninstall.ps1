# Learn more about .exe uninstall scripts:
# http://fleetdm.com/learn-more-about/exe-uninstall-scripts
#
# Removes the machine-wide install and any copy that earlier versions of the
# install script left in the SYSTEM account's profile. If the uninstaller is
# missing or fails, removes the registration, shortcuts and folder itself.

$displayName = "Stretchly"
$publisher = "Jan Hovancik"
$installDirName = "Stretchly"
$exitCode = 0

function Get-Entries {
    $roots = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
        'Registry::HKEY_USERS\S-1-5-18\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'Registry::HKEY_USERS\S-1-5-18\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    )
    foreach ($root in $roots) {
        if (-not (Test-Path -LiteralPath $root)) { continue }
        foreach ($sub in (Get-ChildItem -LiteralPath $root -ErrorAction Stop)) {
            $key = Get-ItemProperty -LiteralPath $sub.PSPath -ErrorAction SilentlyContinue
            if ($key.DisplayName -notlike "$displayName *" -or $key.Publisher -ne $publisher) { continue }
            [PSCustomObject]@{
                KeyPath = $sub.PSPath
                Command = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
            }
        }
    }
}

function Stop-InstallDirProcesses {
    param($Dir)
    $running = @(Get-Process -ErrorAction SilentlyContinue |
        Where-Object { $_.Path -and $_.Path.StartsWith("$Dir\", [System.StringComparison]::OrdinalIgnoreCase) })
    if ($running.Count -gt 0) {
        Write-Host "Stopping $($running.Count) running $displayName process(es)."
        $running | Stop-Process -Force -ErrorAction SilentlyContinue
        $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue
    }
}

function Remove-Entry {
    param($Entry)

    $exePath = $null
    $arguments = ""
    if ($Entry.Command -match '^\s*"([^"]+)"\s*(.*)$') {
        $exePath = $matches[1]
        $arguments = $matches[2].Trim()
    } elseif ($Entry.Command -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $exePath = $matches[1]
        $arguments = $matches[2].Trim()
    }

    $installDir = if ($exePath) { Split-Path $exePath -Parent } else { $null }
    # A copy installed into SYSTEM's profile by the 32-bit installer is under
    # SysWOW64, but its registration names system32.
    if ($installDir -and -not (Test-Path -LiteralPath $installDir)) {
        $redirected = $installDir -replace '(?i)\\system32\\', '\SysWOW64\'
        if (Test-Path -LiteralPath $redirected) {
            $installDir = $redirected
            $exePath = Join-Path $installDir (Split-Path $exePath -Leaf)
        }
    }

    if ($exePath -and (Test-Path -LiteralPath $exePath -PathType Leaf)) {
        Stop-InstallDirProcesses $installDir
        if ($arguments -notmatch '(?i)(^|\s)/S($|\s)') { $arguments = "$arguments /S".Trim() }
        # _?= runs the uninstaller in place, so this waits for it to finish.
        $arguments = "$arguments _?=$installDir"
        Write-Host "Uninstall command: $exePath"
        Write-Host "Uninstall args: $arguments"
        try {
            $process = Start-Process -FilePath $exePath -ArgumentList $arguments -Wait -PassThru -NoNewWindow
            Write-Host "Uninstall exit code: $($process.ExitCode)"
        } catch {
            Write-Host "Error running uninstaller: $_"
        }
    } else {
        Write-Host "Uninstaller not found."
    }

    if (Test-Path -LiteralPath $Entry.KeyPath) {
        Write-Host "$displayName is still registered; removing it manually."
        if ($installDir) { Stop-InstallDirProcesses $installDir }
        Remove-Item -LiteralPath $Entry.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
        $shortcuts = @(
            "$env:ProgramData\Microsoft\Windows\Start Menu\Programs\Stretchly.lnk",
            "$env:PUBLIC\Desktop\Stretchly.lnk",
            "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Stretchly.lnk",
            "$env:USERPROFILE\Desktop\Stretchly.lnk"
        )
        foreach ($shortcut in $shortcuts) {
            $redirected = $shortcut -replace '(?i)\\system32\\', '\SysWOW64\'
            Remove-Item -LiteralPath $shortcut, $redirected -Force -ErrorAction SilentlyContinue
        }
    }

    if ($installDir -and (Split-Path $installDir -Leaf) -eq $installDirName -and (Test-Path -LiteralPath $installDir)) {
        Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Remove-StretchlyFromPath {
    # Uses the raw registry API so unexpanded entries like %SystemRoot% are kept.
    $envKeys = @(
        @{ Hive = [Microsoft.Win32.Registry]::CurrentUser; SubKey = 'Environment' },
        @{ Hive = [Microsoft.Win32.Registry]::LocalMachine; SubKey = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment' }
    )
    foreach ($envKey in $envKeys) {
        try {
            $key = $envKey.Hive.OpenSubKey($envKey.SubKey, $true)
            if (-not $key) { continue }
            $current = $key.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
            if ($current) {
                $kind = $key.GetValueKind('Path')
                $updated = ($current -split ';' | Where-Object { $_ -and $_ -notlike '*\Stretchly\bin' }) -join ';'
                if ($updated -ne $current) {
                    $key.SetValue('Path', $updated, $kind)
                    Write-Host "Removed $displayName from PATH in $($envKey.SubKey)"
                }
            }
            $key.Close()
        } catch {
            Write-Host "PATH cleanup skipped for $($envKey.SubKey): $_"
        }
    }
}

try {
    $entries = @(Get-Entries)
    if ($entries.Count -eq 0) {
        Write-Host "$displayName is not installed."
        Exit 0
    }

    foreach ($entry in $entries) {
        Write-Host "Removing $($entry.KeyPath)"
        Remove-Entry $entry
    }

    Remove-StretchlyFromPath

    if (@(Get-Entries).Count -gt 0) {
        Write-Host "$displayName is still registered after uninstall."
        $exitCode = 1
    } else {
        Write-Host "$displayName is no longer present."
    }
} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
