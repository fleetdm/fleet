# Learn more about .exe uninstall scripts:
# http://fleetdm.com/learn-more-about/exe-uninstall-scripts
#
# Removes the machine-wide install and any copy that earlier versions of the
# install script left in the SYSTEM account's profile.

$displayName = "Twine"
$publisher = "Chris Klimas"
$installDirName = "Twine"
$exitCode = 0

function Get-Entries {
    $roots = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
        'Registry::HKEY_USERS\S-1-5-18\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'Registry::HKEY_USERS\S-1-5-18\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    )
    foreach ($root in $roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if ($key.DisplayName -notlike "$displayName *" -or $key.Publisher -ne $publisher) { continue }
            [PSCustomObject]@{
                KeyPath = $sub.PSPath
                Command = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
            }
        }
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

    # A copy installed into SYSTEM's profile by the 32-bit installer is under
    # SysWOW64, but its registration names system32.
    if ($exePath -and -not (Test-Path -LiteralPath $exePath)) {
        $redirected = $exePath -replace '(?i)\\system32\\', '\SysWOW64\'
        if (Test-Path -LiteralPath $redirected) { $exePath = $redirected }
    }

    $result = 0
    $installDir = $null
    if ($exePath -and (Test-Path -LiteralPath $exePath)) {
        $installDir = Split-Path $exePath -Parent
        $running = @(Get-Process -ErrorAction SilentlyContinue |
            Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) })
        if ($running.Count -gt 0) {
            Write-Host "Stopping $($running.Count) running $displayName process(es)."
            $running | Stop-Process -Force -ErrorAction SilentlyContinue
            $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue
        }

        if ($arguments -notmatch '(?i)(^|\s)/S($|\s)') { $arguments = "$arguments /S".Trim() }
        # _?= runs the uninstaller in place, so this waits for it to finish.
        $arguments = "$arguments _?=$installDir"
        Write-Host "Uninstall command: $exePath"
        Write-Host "Uninstall args: $arguments"
        $process = Start-Process -FilePath $exePath -ArgumentList $arguments -Wait -PassThru -NoNewWindow
        $result = $process.ExitCode
        Write-Host "Uninstall exit code: $result"
    } elseif (-not $exePath) {
        Write-Host "Uninstall command is missing or unrecognized: '$($Entry.Command)'"
        return 1
    } else {
        $leftover = @(@($exePath, ($exePath -replace '(?i)\\system32\\', '\SysWOW64\')) |
            ForEach-Object { Split-Path $_ -Parent } |
            Where-Object { $_ -and (Test-Path -LiteralPath $_) })
        if ($leftover.Count -gt 0) {
            Write-Host "Uninstaller not found but $($leftover[0]) still exists."
            return 1
        }
        Write-Host "Uninstaller and install directory not found; removing the leftover registration."
    }

    if ($result -eq 0) {
        if ($installDir -and (Split-Path $installDir -Leaf) -eq $installDirName -and (Test-Path -LiteralPath $installDir)) {
            Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
            if (Test-Path -LiteralPath $installDir) {
                Write-Host "$installDir still exists after uninstall."
                return 1
            }
        }
        if (Test-Path -Path $Entry.KeyPath) {
            Remove-Item -Path $Entry.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
    return $result
}

try {
    $entries = @(Get-Entries)
    if ($entries.Count -eq 0) {
        Write-Host "$displayName is not installed."
        Exit 0
    }

    foreach ($entry in $entries) {
        Write-Host "Removing $($entry.KeyPath)"
        $result = Remove-Entry $entry
        if ($result -ne 0 -and $exitCode -eq 0) { $exitCode = $result }
    }

    if (@(Get-Entries).Count -gt 0) {
        Write-Host "$displayName is still registered after uninstall."
        if ($exitCode -eq 0) { $exitCode = 1 }
    }
} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
