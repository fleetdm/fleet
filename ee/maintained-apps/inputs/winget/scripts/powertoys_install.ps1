# PowerToys is managed by Fleet as a PER-MACHINE (HKLM) install. Windows also
# ships a per-user installer (PowerToysUserSetup), so a host may already have a
# stale per-user copy. Fleet's patch policy is scope-blind (osquery's "programs"
# table reads HKLM + every loaded user hive), so a lingering per-user copy keeps
# the policy red and leaves two copies on disk.
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

$ExpectedExitCodes = @(0, 1641, 3010, 1223)

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
    param(
        [Parameter(Mandatory = $true)][string]$DisplayNameLike,
        [string]$PublisherLike = ''
    )

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
            if ($PublisherLike -ne '' -and $key.Publisher -notlike $PublisherLike) { continue }

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

            # PowerToys installers are WiX Burn bundles. Ensure a quiet uninstall.
            if ($exe -match '(?i)msiexec') {
                $uninstallArgs = $uninstallArgs -replace '(?i)/i(\{)', '/x$1'
                if ($uninstallArgs -notmatch '(?i)/x') { $uninstallArgs = ("/x $uninstallArgs").Trim() }
                if ($uninstallArgs -notmatch '(?i)/qn') { $uninstallArgs = ("$uninstallArgs /qn").Trim() }
                if ($uninstallArgs -notmatch '(?i)/norestart') { $uninstallArgs = ("$uninstallArgs /norestart").Trim() }
            } else {
                if ($uninstallArgs -notmatch '(?i)/uninstall') { $uninstallArgs = ("$uninstallArgs /uninstall").Trim() }
                if ($uninstallArgs -notmatch '(?i)/quiet') { $uninstallArgs = ("$uninstallArgs /quiet").Trim() }
                if ($uninstallArgs -notmatch '(?i)/norestart') { $uninstallArgs = ("$uninstallArgs /norestart").Trim() }
            }

            if (-not ($exe -match '(?i)msiexec') -and -not (Test-Path -LiteralPath $exe)) {
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
    Stop-Process -Name "PowerToys" -Force -ErrorAction SilentlyContinue
    Remove-OtherScopeCopies -DisplayNameLike "PowerToys*" -PublisherLike "Microsoft Corporation*"
} catch {
    # Cleanup is best-effort; proceed to install regardless.
    Write-Host "Warning during per-user cleanup: $_"
}

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/install /quiet /norestart"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
