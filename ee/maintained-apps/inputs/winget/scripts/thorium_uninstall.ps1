# Learn more about .exe uninstall scripts:
# http://fleetdm.com/learn-more-about/exe-uninstall-scripts
#
# Removes the machine-wide install and any copy that earlier versions of the
# install script left in the SYSTEM account's profile.

$displayName = "Thorium"
$publisher = "EDRLab"
$installDirName = "Thorium"
$exitCode = 0

function Get-Entries {
    $roots = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
        'Registry::HKEY_USERS\S-1-5-18\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'Registry::HKEY_USERS\S-1-5-18\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    )
    foreach ($root in $roots) {
        # A missing root is fine; any other read failure stops the uninstall.
        try {
            $subs = @(Get-ChildItem -Path $root -ErrorAction Stop)
        } catch [System.Management.Automation.ItemNotFoundException] {
            continue
        }
        foreach ($sub in $subs) {
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
    if ($exePath -and -not (Test-Path -LiteralPath (Split-Path $exePath -Parent))) {
        $exePath = $exePath -replace '(?i)\\system32\\', '\SysWOW64\'
    }
    $installDir = if ($exePath) { Split-Path $exePath -Parent } else { $null }

    if (-not $installDir) {
        Write-Host "Couldn't find the uninstaller in: $($Entry.Command)"
        return 1
    }

    $isThoriumDir = (Split-Path $installDir -Leaf) -eq $installDirName
    $running = @()
    if ($isThoriumDir) {
        $running = @(Get-Process -ErrorAction SilentlyContinue |
            Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) })
    }
    if ($running.Count -gt 0) {
        Write-Host "Stopping $($running.Count) running $displayName process(es)."
        $running | Stop-Process -Force -ErrorAction SilentlyContinue
        $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue
        if (@($running | Where-Object { -not $_.HasExited }).Count -gt 0) {
            Write-Host "$displayName is still running after 30 seconds."
            return 1
        }
    }

    $result = 0
    $ran = $false
    if (Test-Path -LiteralPath $exePath -PathType Leaf) {
        if ($arguments -notmatch '(?i)(^|\s)/S($|\s)') { $arguments = "$arguments /S".Trim() }
        # _?= runs the uninstaller in place, so this waits for it to finish.
        $arguments = "$arguments _?=$installDir"
        Write-Host "Uninstall command: $exePath"
        Write-Host "Uninstall args: $arguments"
        try {
            $process = Start-Process -FilePath $exePath -ArgumentList $arguments -Wait -PassThru -NoNewWindow
            $result = $process.ExitCode
            $ran = $true
            Write-Host "Uninstall exit code: $result"
        } catch {
            Write-Host "Couldn't run the uninstaller: $_"
        }
    }
    if (-not $ran) {
        # Keep the registration unless the leftover folder is gone, so a retry can finish.
        if (Test-Path -LiteralPath $installDir) {
            if (-not $isThoriumDir) {
                Write-Host "Uninstaller not available; not removing unexpected folder $installDir."
                return 1
            }
            Write-Host "Uninstaller not available; removing $installDir."
            # Files of just-stopped processes can stay locked for a moment.
            for ($i = 0; $i -lt 10 -and (Test-Path -LiteralPath $installDir); $i++) {
                if ($i -gt 0) { Start-Sleep -Seconds 2 }
                Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
            }
            if (Test-Path -LiteralPath $installDir) {
                Write-Host "Couldn't remove $installDir."
                return 1
            }
        } else {
            Write-Host "Uninstaller not available; removing the leftover registration."
        }
    }

    if ($result -eq 0) {
        if ($ran -and $isThoriumDir -and (Test-Path -LiteralPath $installDir)) {
            Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
            if (Test-Path -LiteralPath $installDir) { Write-Host "Some files in $installDir couldn't be removed." }
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
