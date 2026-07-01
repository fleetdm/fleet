# GIMP is managed by Fleet as a PER-MACHINE install (Inno Setup /ALLUSERS).
# GIMP also offers a per-user install, so a host may already have a stale per-user
# copy. Fleet's patch policy is scope-blind (osquery's "programs" table reads HKLM
# + every loaded user hive), so a lingering per-user copy keeps the policy red and
# leaves two copies on disk.
#
# Pattern A (remove-and-replace): before installing the machine copy, remove any
# per-user copy so the device converges on a single canonical copy. The machine
# installer upgrades an existing machine copy in place, so same-scope data is
# preserved; only the cross-scope (per-user) copy is removed.
# See https://github.com/fleetdm/fleet/issues/48248.
#
# NOTE: Fleet runs install scripts as SYSTEM, where HKCU maps to SYSTEM's own
# hive — NOT the logged-on user's. Per-user copies must be found under
# HKEY_USERS\<user SID>, which is what Remove-OtherScopeCopies does below.
# Removal is best-effort: it never aborts the machine install, and a copy that
# survives keeps the (truthful) scope-blind policy red rather than false-green.

# Match GIMP 3.x only (the FMA targets GIMP.GIMP.3); avoids touching GIMP 2.
$displayNameLike = "GIMP 3*"

function Get-UninstallExeAndArgs {
    param([string]$Command)
    # Registry uninstall strings come in three shapes; parse defensively.
    if ($Command -match '^\s*"([^"]+)"\s*(.*)$') {
        return @{ Exe = $Matches[1]; Args = $Matches[2].Trim() }
    } elseif ($Command -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        return @{ Exe = $Matches[1]; Args = $Matches[2].Trim() }
    } elseif ($Command -match '^\s*(\S+)\s*(.*)$') {
        return @{ Exe = $Matches[1]; Args = $Matches[2].Trim() }
    }
    return $null
}

function Remove-OtherScopeCopies {
    param([Parameter(Mandatory = $true)][string]$DisplayNameLike)

    # Per-user uninstall registrations live in the logged-on users' hives.
    $roots = @()
    foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
        if ($hive.Name -match '_Classes$') { continue }
        # Real interactive users only (skip .DEFAULT and service SIDs).
        if ($hive.PSChildName -notmatch '^S-1-5-21-') { continue }
        $roots += "Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall"
        $roots += "Registry::$($hive.Name)\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall"
    }

    foreach ($root in $roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if (-not $key.DisplayName) { continue }
            if ($key.DisplayName -notlike $DisplayNameLike) { continue }

            $command = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
            if (-not $command) {
                Write-Host "Per-user copy '$($key.DisplayName)' has no uninstall string; skipping."
                continue
            }

            $parsed = Get-UninstallExeAndArgs $command
            if (-not $parsed) {
                Write-Host "Could not parse uninstall string for '$($key.DisplayName)': $command"
                continue
            }

            $exe = $parsed.Exe
            $uninstallArgs = $parsed.Args

            # GIMP uses an Inno Setup uninstaller; ensure a silent uninstall.
            if ($uninstallArgs -notmatch '(?i)/VERYSILENT') { $uninstallArgs = ("$uninstallArgs /VERYSILENT").Trim() }
            if ($uninstallArgs -notmatch '(?i)/SUPPRESSMSGBOXES') { $uninstallArgs = ("$uninstallArgs /SUPPRESSMSGBOXES").Trim() }
            if ($uninstallArgs -notmatch '(?i)/NORESTART') { $uninstallArgs = ("$uninstallArgs /NORESTART").Trim() }

            if (-not (Test-Path -LiteralPath $exe)) {
                Write-Host "Per-user uninstaller missing on disk for '$($key.DisplayName)': $exe"
                continue
            }

            Write-Host "Removing per-user copy: '$($key.DisplayName)'"
            Write-Host "  Command: $exe"
            Write-Host "  Args: $uninstallArgs"
            try {
                $opts = @{ FilePath = $exe; PassThru = $true; Wait = $true; NoNewWindow = $true }
                if ($uninstallArgs -ne '') { $opts.ArgumentList = $uninstallArgs }
                $p = Start-Process @opts
                Write-Host "  Per-user uninstall exit code: $($p.ExitCode)"
            } catch {
                # Best effort: never fail the machine install because of other-scope cleanup.
                Write-Host "  WARNING: failed to remove per-user copy: $_"
            }
        }
    }
}

try {
    Remove-OtherScopeCopies -DisplayNameLike $displayNameLike
} catch {
    # Cleanup is best-effort; proceed to install regardless.
    Write-Host "Warning during per-user cleanup: $_"
}

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Inno Setup installer with /ALLUSERS for machine-scope installation
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /ALLUSERS"
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# Prints the exit code
Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
