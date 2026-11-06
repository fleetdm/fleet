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
    if ($exePath -and -not (Test-Path -LiteralPath (Split-Path $exePath -Parent))) {
        $exePath = $exePath -replace '(?i)\\system32\\', '\SysWOW64\'
    }
    $installDir = if ($exePath) { Split-Path $exePath -Parent } else { $null }

    $result = 0
    $ran = $false
    if ($exePath -and (Test-Path -LiteralPath $exePath -PathType Leaf)) {
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
        Write-Host "Uninstaller not available; removing the leftover files and registration."
    }

    if ($result -eq 0) {
        if ($installDir -and (Split-Path $installDir -Leaf) -eq $installDirName -and (Test-Path -LiteralPath $installDir)) {
            Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
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
