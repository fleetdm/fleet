# python.org's Windows installer is a WiX "Burn" bundle. A Python 3.14 install
# registers many visible uninstall entries:
#   * the bundle:    "Python 3.14.<patch> (64-bit)" -> a cached Burn EXE
#   * components:    "Python 3.14.<patch> Core Interpreter (64-bit)",
#                    "...Standard Library (64-bit)", etc. -> individual MSIs
#
# The bundle's "/uninstall /quiet" relaunches a cached copy of itself in a clean
# room and removes the components asynchronously, which is slow/racy to wait on.
# Instead we remove the component MSIs directly with "msiexec /X ... /qn" (fully
# synchronous), then clear the bundle's own registration. We match on the
# DisplayName (anchored for the bundle) rather than the Publisher string, and
# scan every hive osquery's "programs" table reads, so this works for both
# per-machine (SYSTEM, HKLM) and per-user (HKU) installs.

$pythonNameLike    = "Python 3.14.* (64-bit)"
$bundleNamePattern = '^Python 3\.14\.\d+ \(64-bit\)$'

# 0 = success, 1605 = not installed, 1614 = already uninstalled, 1641 = reboot
# initiated, 3010 = reboot required.
$ExpectedExitCodes = @(0, 1605, 1614, 1641, 3010)
$exitCode = 0

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
    param([string[]]$Roots)
    $list = @()
    foreach ($root in $Roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if (-not $key.DisplayName) { continue }
            if ($key.DisplayName -notlike $pythonNameLike) { continue }
            $list += [PSCustomObject]@{
                DisplayName    = $key.DisplayName
                KeyPath        = $sub.PSPath
                KeyName        = $sub.PSChildName
                Uninstall      = $key.UninstallString
                QuietUninstall = $key.QuietUninstallString
            }
        }
    }
    return $list
}

# Returns an MSI product code ({GUID}) for an entry, or $null if it isn't an MSI.
function Get-MsiProductCode {
    param($Entry)
    if ($Entry.Uninstall -match "(?i)MsiExec\.exe\s+/[IX]\s*(\{[0-9A-F-]+\})") { return $Matches[1] }
    if ($Entry.KeyName -match "(?i)^\{[0-9A-F-]+\}$") { return $Entry.KeyName }
    return $null
}

function Get-ExePath {
    param([string]$Command)
    if ($Command -match '^\s*"([^"]+)"') { return $Matches[1] }
    if ($Command -match '(?i)^\s*(.+?\.exe)(\s|$)') { return $Matches[1] }
    return $null
}

try {

    $roots = Get-UninstallRoots
    $entries = Get-PythonEntries -Roots $roots
    if ($entries.Count -eq 0) {
        Write-Host "No Python 3.14 (64-bit) entries found (already removed)."
        Exit 0
    }

    # Stop any running interpreter/launcher so component MSIs aren't locked.
    foreach ($proc in @("python", "pythonw", "py", "pyw")) {
        Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
    }

    # --- Phase 1: remove component MSIs directly (synchronous). ---
    foreach ($e in $entries) {
        if ($e.DisplayName -match $bundleNamePattern) { continue } # handle the bundle in phase 2
        $code = Get-MsiProductCode $e
        if (-not $code) { Write-Host "Skipping non-MSI component: '$($e.DisplayName)'"; continue }

        Write-Host "Removing component: '$($e.DisplayName)' ($code)"
        $p = Start-Process -FilePath "msiexec.exe" -ArgumentList "/X $code /qn /norestart" -PassThru -Wait
        Write-Host "  Exit code: $($p.ExitCode)"
        if (($ExpectedExitCodes -notcontains $p.ExitCode) -and ($exitCode -eq 0)) { $exitCode = $p.ExitCode }
    }

    # --- Phase 2: remove the bundle. With its components gone, the bundle's own
    # uninstall is quick; if its cached EXE is missing or it leaves an ARP entry,
    # delete the orphaned registration so detection clears. ---
    foreach ($e in (Get-PythonEntries -Roots $roots | Where-Object { $_.DisplayName -match $bundleNamePattern })) {
        $command = if ($e.QuietUninstall) { $e.QuietUninstall } else { $e.Uninstall }
        $exe = if ($command) { Get-ExePath $command } else { $null }

        if ($exe -and (Test-Path -LiteralPath $exe)) {
            Write-Host "Uninstalling bundle: '$($e.DisplayName)' via $exe"
            $p = Start-Process -FilePath $exe -ArgumentList "/uninstall /quiet /norestart" -PassThru -Wait
            Write-Host "  Exit code: $($p.ExitCode)"
            if (($ExpectedExitCodes -notcontains $p.ExitCode) -and ($exitCode -eq 0)) { $exitCode = $p.ExitCode }
        }

        # If a registration still lingers (e.g. orphaned bundle entry), remove it.
        $still = Get-ItemProperty $e.KeyPath -ErrorAction SilentlyContinue
        if ($still -and $still.DisplayName) {
            Write-Host "Removing leftover bundle registration: '$($e.DisplayName)'"
            Remove-Item -Path $e.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    # --- Verify nothing remains in the programs table. ---
    $remaining = Get-PythonEntries -Roots $roots
    if ($remaining.Count -gt 0) {
        Write-Host "WARNING: $($remaining.Count) Python 3.14 entry(ies) still present after uninstall:"
        $remaining | ForEach-Object { Write-Host "  - $($_.DisplayName) [$($_.KeyPath)]" }
        if ($exitCode -eq 0) { $exitCode = 1 }
    } else {
        Write-Host "All Python 3.14 (64-bit) entries removed."
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

if ($ExpectedExitCodes -contains $exitCode) { Exit 0 } else { Exit $exitCode }
