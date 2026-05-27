# Fleet's validator finds the app via osquery's "programs" table, which reads
# uninstall entries from HKLM, Wow6432Node, AND per-user HKEY_USERS hives.
# Power BI registers more than one entry with inconsistent name spacing
# ("Power BI Desktop" vs "PowerBI Desktop"), so we scan every hive, match
# space-insensitively, and uninstall EVERY matching entry.

# 0 = success; 1605 = product not installed; 1641/3010 = reboot required.
$ExpectedExitCodes = @(0, 1605, 1641, 3010)
$exitCode = 0

try {

    # Build the list of Uninstall roots across all hives.
    $roots = [System.Collections.Generic.List[string]]::new()
    $roots.Add('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall')
    $roots.Add('HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall')
    foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
        if ($hive.Name -match '_Classes$') { continue }
        $roots.Add("Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall")
        $roots.Add("Registry::$($hive.Name)\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall")
    }

    # Stop running Power BI processes so locked files don't block the uninstalls.
    foreach ($proc in @("PBIDesktop", "msmdsrv", "Microsoft.Mashup.Container")) {
        Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
    }

    $found = $false
    foreach ($root in $roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if (-not $key.DisplayName) { continue }

            $normalized = ($key.DisplayName -replace '\s', '').ToLower()
            if (-not $normalized.Contains("powerbidesktop")) { continue }

            $found = $true
            Write-Host "Found entry: '$($key.DisplayName)' under $root"

            $uninstallCommand = if ($key.QuietUninstallString) {
                $key.QuietUninstallString
            } else {
                $key.UninstallString
            }

            $uninstallCmd  = $null
            $uninstallArgs = $null

            if ($uninstallCommand -match "(?i)MsiExec\.exe\s+/[IX]\s*(\{[A-F0-9-]+\})") {
                # MSI-backed entry.
                $uninstallCmd  = "MsiExec.exe"
                $uninstallArgs = "/X $($Matches[1]) /qn /norestart"
            } elseif (-not $uninstallCommand -and ($sub.PSChildName -match "(?i)^\{[A-F0-9-]+\}$")) {
                # No uninstall string, but the key name is an MSI product code GUID.
                $uninstallCmd  = "MsiExec.exe"
                $uninstallArgs = "/X $($sub.PSChildName) /qn /norestart"
            } elseif ($uninstallCommand) {
                # Plain EXE uninstaller. Split the quoted command from its args.
                $splitArgs = $uninstallCommand.Split('"')
                if ($splitArgs.Length -gt 1) {
                    $uninstallCmd = $splitArgs[1]
                    if ($splitArgs.Length -eq 3 -and $splitArgs[2].Trim()) {
                        $uninstallArgs = "$($splitArgs[2].Trim()) /S".Trim()
                    } else {
                        $uninstallArgs = "/S"
                    }
                } else {
                    $uninstallCmd  = $uninstallCommand
                    $uninstallArgs = "/S"
                }
            } else {
                Write-Host "  No usable uninstall command, skipping."
                continue
            }

            Write-Host "  Command: $uninstallCmd"
            Write-Host "  Args: $uninstallArgs"

            $opts = @{ FilePath = $uninstallCmd; PassThru = $true; Wait = $true }
            if ($uninstallArgs) { $opts.ArgumentList = $uninstallArgs }

            $process = Start-Process @opts
            $entryExit = $process.ExitCode
            Write-Host "  Exit code: $entryExit"

            # msiexec can return before fully finished; wait it out.
            $elapsed = 0
            while ((Get-Process -Name "msiexec" -ErrorAction SilentlyContinue) -and ($elapsed -lt 120)) {
                Start-Sleep -Seconds 2
                $elapsed += 2
            }

            # Record the first failing (non-expected) exit code, but keep going.
            if (($ExpectedExitCodes -notcontains $entryExit) -and ($exitCode -eq 0)) {
                $exitCode = $entryExit
            }
        }
    }

    if (-not $found) {
        Write-Host "No Power BI Desktop uninstall entries found (already removed)."
        $exitCode = 0
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

if ($ExpectedExitCodes -contains $exitCode) {
    Exit 0
} else {
    Exit $exitCode
}
