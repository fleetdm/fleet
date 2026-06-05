# python.org's Windows installer is a WiX "Burn" bundle. A per-machine x64
# install registers MANY visible uninstall entries:
#   * the bundle:    "Python 3.14.<patch> (64-bit)" -> a cached Burn EXE run with
#                    "/uninstall" (removes the bundle and all of its components)
#   * components:    "Python 3.14.<patch> Core Interpreter (64-bit)",
#                    "...Standard Library (64-bit)", etc. -> individual MSIs
#
# Strategy (mirrors the Power BI Burn uninstaller):
#   Phase 1: uninstall the bundle via its EXE. Burn relaunches a cached copy of
#            itself and drives msiexec asynchronously, so wait for those to exit.
#   Phase 2: sweep any component MSIs the bundle left behind via msiexec /X.
# We match on the DisplayName only (anchored for the bundle) rather than on the
# Publisher string, and scan every hive osquery's "programs" table reads, so the
# uninstall doesn't silently no-op on a publisher/registry-view mismatch.

# Catches the bundle and every component entry.
$pythonNameLike = "Python 3.14.* (64-bit)"
# Anchored: matches ONLY the bundle, e.g. "Python 3.14.5 (64-bit)" — not
# components such as "Python 3.14.5 Core Interpreter (64-bit)".
$bundleNamePattern = '^Python 3\.14\.\d+ \(64-bit\)$'

# 0 = success, 1605 = product not installed, 1614 = product already uninstalled,
# 1641 = success (reboot initiated), 3010 = success (reboot required).
$ExpectedExitCodes = @(0, 1605, 1614, 1641, 3010)
$exitCode = 0

function Wait-ForProcessExit {
    param([string[]]$Names, [int]$TimeoutSeconds = 240)
    $elapsed = 0
    while ($elapsed -lt $TimeoutSeconds) {
        $running = $Names | Where-Object { Get-Process -Name $_ -ErrorAction SilentlyContinue }
        if (-not $running) { break }
        Start-Sleep -Seconds 3
        $elapsed += 3
    }
}

function Get-UninstallRoots {
    $roots = [System.Collections.Generic.List[string]]::new()
    $roots.Add('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall')
    $roots.Add('HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall')
    foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
        if ($hive.Name -match '_Classes$') { continue }
        $roots.Add("Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall")
        $roots.Add("Registry::$($hive.Name)\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall")
    }
    return $roots
}

function Get-PythonEntries {
    param([string[]]$Roots, [string]$NameLike)
    $list = @()
    foreach ($root in $Roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if (-not $key.DisplayName) { continue }
            if ($key.DisplayName -notlike $NameLike) { continue }
            $list += [PSCustomObject]@{
                DisplayName  = $key.DisplayName
                KeyName      = $sub.PSChildName
                Uninstall    = $key.UninstallString
                QuietUninstall = $key.QuietUninstallString
            }
        }
    }
    return $list
}

function Get-ExePath {
    param([string]$Command)
    if ($Command -match '^\s*"([^"]+)"') { return $Matches[1] }
    if ($Command -match '(?i)^\s*(.+?\.exe)(\s|$)') { return $Matches[1] }
    return $null
}

try {

    $roots = Get-UninstallRoots
    $entries = Get-PythonEntries -Roots $roots -NameLike $pythonNameLike
    if ($entries.Count -eq 0) {
        Write-Host "No Python 3.14 (64-bit) entries found (already removed)."
        Exit 0
    }

    # Best-effort: stop any running interpreter so component MSIs aren't locked.
    foreach ($proc in @("python", "pythonw", "py")) {
        Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
    }

    # --- Phase 1: uninstall the Burn bundle(s) FIRST. ---
    foreach ($e in ($entries | Where-Object { $_.DisplayName -match $bundleNamePattern })) {
        $command = if ($e.QuietUninstall) { $e.QuietUninstall } else { $e.Uninstall }
        if (-not $command) { Write-Host "No uninstall string for bundle '$($e.DisplayName)'"; continue }

        $exe = Get-ExePath $command
        if (-not $exe) { Write-Host "Could not parse bundle exe from: $command"; continue }
        if (-not (Test-Path -LiteralPath $exe)) { Write-Host "Bundle exe missing: $exe"; continue }

        Write-Host "Uninstalling bundle: '$($e.DisplayName)'"
        Write-Host "  Command: $exe"
        Write-Host "  Args: /uninstall /quiet /norestart"
        $p = Start-Process -FilePath $exe -ArgumentList "/uninstall /quiet /norestart" -PassThru -Wait
        Write-Host "  Exit code: $($p.ExitCode)"
        if (($ExpectedExitCodes -notcontains $p.ExitCode) -and ($exitCode -eq 0)) { $exitCode = $p.ExitCode }

        # Burn relaunches a cached copy of itself and spawns msiexec asynchronously.
        # Give the relaunch a moment to appear, then wait for it (and msiexec) to finish.
        Start-Sleep -Seconds 3
        $exeName = [System.IO.Path]::GetFileNameWithoutExtension($exe)
        Wait-ForProcessExit -Names @($exeName, "msiexec") -TimeoutSeconds 300
    }

    # --- Phase 2: remove any component MSIs the bundle didn't clean up. ---
    foreach ($e in (Get-PythonEntries -Roots $roots -NameLike $pythonNameLike)) {
        $msiCode = $null
        if ($e.Uninstall -match "(?i)MsiExec\.exe\s+/[IX]\s*(\{[A-F0-9-]+\})") { $msiCode = $Matches[1] }
        elseif ($e.KeyName -match "(?i)^\{[A-F0-9-]+\}$") { $msiCode = $e.KeyName }
        if (-not $msiCode) {
            Write-Host "Skipping non-MSI leftover: '$($e.DisplayName)' ($($e.Uninstall))"
            continue
        }

        Write-Host "Removing leftover MSI: '$($e.DisplayName)' ($msiCode)"
        $p = Start-Process -FilePath "msiexec.exe" -ArgumentList "/X $msiCode /qn /norestart" -PassThru -Wait
        Write-Host "  Exit code: $($p.ExitCode)"
        Wait-ForProcessExit -Names @("msiexec") -TimeoutSeconds 240
        if (($ExpectedExitCodes -notcontains $p.ExitCode) -and ($exitCode -eq 0)) { $exitCode = $p.ExitCode }
    }

    # Verify nothing remains in the programs table.
    $remaining = Get-PythonEntries -Roots $roots -NameLike $pythonNameLike
    if ($remaining.Count -gt 0) {
        Write-Host "WARNING: $($remaining.Count) Python 3.14 entry(ies) still present after uninstall:"
        $remaining | ForEach-Object { Write-Host "  - $($_.DisplayName)" }
        if ($exitCode -eq 0) { $exitCode = 1 }
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

if ($ExpectedExitCodes -contains $exitCode) { Exit 0 } else { Exit $exitCode }
