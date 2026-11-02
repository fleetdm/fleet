# python.org's Windows installer is a WiX "Burn" bundle. A Python 3.13 install
# registers many visible uninstall entries:
#   * the bundle:    "Python 3.13.<patch> (ARM64)" -> a cached Burn EXE
#   * components:    "Python 3.13.<patch> Core Interpreter (ARM64)",
#                    "...Standard Library (ARM64)", "...Executables (ARM64)",
#                    etc. -> individual MSIs
#
# The component MSIs must be removed in DEPENDENCY ORDER: the feature components
# (Standard Library, Tcl/Tk, pip, docs, etc.) run uninstall custom actions that
# invoke python.exe, so the "Executables" and "Core Interpreter" components that
# provide python.exe must be removed LAST. Removing them first makes the rest
# fail with exit code 1603. This ordering + "REBOOT=ReallySuppress /qn" mirrors
# the documented silentinstallhq.com Python uninstall.
#
# We match on the DisplayName (anchored for the bundle) rather than the Publisher
# string. Only entries in HKLM, SYSTEM's hive and the hive of the account running
# this script (SYSTEM under Fleet) are removed: another user's hive is writable by
# that user, so its uninstaller or MSI product code would run as SYSTEM. The bundle
# registers in the installing account's hive even for an all-users install. Other
# user hives (which osquery's "programs" table also reads) are checked at the end
# so a per-user install is reported instead of passing silently.

$pythonNameLike    = "Python 3.13.* (ARM64)"
$bundleNamePattern = '^Python 3\.13\.\d+ \(ARM64\)$'
$trustedSids       = @('S-1-5-18', [System.Security.Principal.WindowsIdentity]::GetCurrent().User.Value)

# 0 = success, 1605 = not installed, 1614 = already uninstalled, 1641 = reboot
# initiated, 3010 = reboot required.
$ExpectedExitCodes = @(0, 1605, 1614, 1641, 3010)
$exitCode = 0

function Get-UninstallRoots {
    param([switch]$IncludeUserHives)
    $roots = [System.Collections.Generic.List[string]]::new()
    $roots.Add('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall')
    $roots.Add('HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall')
    # Trusted hives, plus user hives (S-1-5-21 local or AD, S-1-12-1 Entra ID) when asked.
    foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
        $sid = $hive.PSChildName
        $isUserHive = $IncludeUserHives -and ($sid -match '^S-1-(5-21|12-1)-[\d-]+$')
        if (($trustedSids -notcontains $sid) -and -not $isUserHive) { continue }
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

# Dependency removal order: feature components first (0), then Executables (1),
# then Core Interpreter LAST (2).
function Get-RemovalPriority {
    param([string]$Name)
    if ($Name -match '(?i)Core Interpreter') { return 2 }
    if ($Name -match '(?i)Executables')      { return 1 }
    return 0
}

function Get-ExePath {
    param([string]$Command)
    if ($Command -match '^\s*"([^"]+)"') { return $Matches[1] }
    if ($Command -match '(?i)^\s*(.+?\.exe)(\s|$)') { return $Matches[1] }
    return $null
}

try {

    $roots = Get-UninstallRoots
    $allRoots = Get-UninstallRoots -IncludeUserHives
    if (@(Get-PythonEntries -Roots $allRoots).Count -eq 0) {
        Write-Host "No Python 3.13 (ARM64) entries found (already removed)."
        Exit 0
    }

    # Stop only running interpreters that belong to THIS Python install so the
    # component MSIs aren't locked. We scope the kill to processes whose image
    # lives under the version-specific "Python313-arm64" directory; matching by image
    # base name alone (python.exe/pythonw.exe/...) would also terminate a
    # side-by-side 3.12/3.14, Microsoft Store, or Anaconda interpreter -- a
    # 3.13 uninstall running as SYSTEM could kill an unrelated long-running
    # workload mid-write. The shared "py"/"pyw" launcher lives in C:\Windows and
    # is version-agnostic, so it is intentionally left running.
    $installDirPattern = '*\Python313-arm64\*'
    foreach ($proc in @("python", "pythonw", "py", "pyw")) {
        Get-Process -Name $proc -ErrorAction SilentlyContinue |
            Where-Object { $_.Path -like $installDirPattern } |
            Stop-Process -Force -ErrorAction SilentlyContinue
    }

    # --- Remove component MSIs in dependency order (Core Interpreter last). ---
    $components = Get-PythonEntries -Roots $roots |
        Where-Object { $_.DisplayName -notmatch $bundleNamePattern } |
        Sort-Object @{ Expression = { Get-RemovalPriority $_.DisplayName } }

    foreach ($e in $components) {
        $code = Get-MsiProductCode $e
        if (-not $code) { Write-Host "Skipping non-MSI component: '$($e.DisplayName)'"; continue }

        Write-Host "Removing component: '$($e.DisplayName)' ($code)"
        $p = Start-Process -FilePath "msiexec.exe" `
            -ArgumentList "/x $code REBOOT=ReallySuppress /qn /norestart" -PassThru -Wait
        Write-Host "  Exit code: $($p.ExitCode)"
        if (($ExpectedExitCodes -notcontains $p.ExitCode) -and ($exitCode -eq 0)) { $exitCode = $p.ExitCode }
    }

    # --- Remove the bundle. With its components gone its own uninstall is quick.
    # A registration left behind with no components is orphaned (its cached EXE
    # is gone, or it outlived a successful uninstall), so delete it to clear
    # detection. After a failed uninstall, keep it so detection and a retry see it. ---
    foreach ($e in @(Get-PythonEntries -Roots $roots | Where-Object { $_.DisplayName -match $bundleNamePattern })) {
        $command = if ($e.QuietUninstall) { $e.QuietUninstall } else { $e.Uninstall }
        $exe = if ($command) { Get-ExePath $command } else { $null }
        $bundleFailed = $false
        if ($exe -and (Test-Path -LiteralPath $exe)) {
            Write-Host "Uninstalling bundle: '$($e.DisplayName)' via $exe"
            $p = Start-Process -FilePath $exe -ArgumentList "/uninstall /quiet /norestart" -PassThru -Wait -ErrorAction Stop
            Write-Host "  Exit code: $($p.ExitCode)"
            if ($ExpectedExitCodes -notcontains $p.ExitCode) {
                $bundleFailed = $true
                if ($exitCode -eq 0) { $exitCode = $p.ExitCode }
            }
        }
        $componentsLeft = @(Get-PythonEntries -Roots $roots | Where-Object { $_.DisplayName -notmatch $bundleNamePattern }).Count -gt 0
        $still = Get-ItemProperty $e.KeyPath -ErrorAction SilentlyContinue
        if ($still -and $still.DisplayName -and -not $bundleFailed -and -not $componentsLeft) {
            Write-Host "Removing orphaned bundle registration: '$($e.DisplayName)'"
            Remove-Item -Path $e.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    # --- Verify nothing remains in the programs table. ---
    $remaining = @(Get-PythonEntries -Roots $allRoots)
    if ($remaining.Count -gt 0) {
        Write-Host "WARNING: $($remaining.Count) Python 3.13 entry(ies) still present after uninstall:"
        $remaining | ForEach-Object { Write-Host "  - $($_.DisplayName) [$($_.KeyPath)]" }
        if (@($remaining | Where-Object { ($_.KeyPath -match 'HKEY_USERS\\(S-1-(5-21|12-1)-[\d-]+)\\') -and ($trustedSids -notcontains $Matches[1]) }).Count -gt 0) {
            Write-Host "Entries under a user's hive are per-user installs, which must be removed as that user."
        }
        if ($exitCode -eq 0) { $exitCode = 1 }
    } else {
        Write-Host "All Python 3.13 (ARM64) entries removed."
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

if ($ExpectedExitCodes -contains $exitCode) { Exit 0 } else { Exit $exitCode }
