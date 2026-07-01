# Fleet Pattern A: converge this app on a single install scope
# (https://github.com/fleetdm/fleet/issues/48248).
#
# Fleet manages this app at "machine" scope. Windows also offers the opposite
# scope, so a host may already have a stale copy there. The patch policy is
# scope-blind (osquery's "programs" table reads HKLM + every loaded user hive),
# so a lingering opposite-scope copy keeps the policy red and leaves two copies
# on disk. Remove the opposite-scope copy before installing the managed copy so
# the device converges on one canonical copy; a same-scope upgrade is left to the
# installer below (preserves data).
#
# Fleet runs install scripts as SYSTEM, where HKCU maps to SYSTEM's own hive --
# NOT the logged-on user's -- so per-user copies are found under HKEY_USERS.
# Removal is best-effort: it never aborts the install, and a copy that survives
# keeps the (truthful) scope-blind policy red rather than false-green.

$fmaTargetScope           = "machine"
$fmaDisplayNameLike       = "Vivaldi*"
$fmaPublisherLike         = ""
$fmaFallbackUninstallArgs = "--uninstall --force-uninstall"

function Get-FmaUninstallExeAndArgs {
    param([string]$Command)
    # Registry uninstall strings come in three shapes; parse defensively.
    if ($Command -match '^\s*"([^"]+)"\s*(.*)$') { return @{ Exe = $Matches[1]; Args = $Matches[2].Trim() } }
    elseif ($Command -match '(?i)^\s*(.+?\.exe)\s*(.*)$') { return @{ Exe = $Matches[1]; Args = $Matches[2].Trim() } }
    elseif ($Command -match '^\s*(\S+)\s*(.*)$') { return @{ Exe = $Matches[1]; Args = $Matches[2].Trim() } }
    return $null
}

function Remove-FmaOtherScopeCopies {
    param(
        [string]$TargetScope,
        [string]$DisplayNameLike,
        [string]$PublisherLike,
        [string]$FallbackArgs
    )

    # Scan ONLY the opposite scope's uninstall hives, so a broad DisplayName match
    # can't touch the copy Fleet manages.
    $roots = @()
    if ($TargetScope -eq 'machine') {
        foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
            if ($hive.Name -match '_Classes$') { continue }
            # Real interactive users only (skip .DEFAULT and service SIDs).
            if ($hive.PSChildName -notmatch '^S-1-5-21-') { continue }
            $roots += "Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall"
            $roots += "Registry::$($hive.Name)\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall"
        }
    } else {
        $roots += 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
        $roots += 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    }

    foreach ($root in $roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if (-not $key.DisplayName) { continue }
            if ($key.DisplayName -notlike $DisplayNameLike) { continue }
            if ($PublisherLike -ne '' -and $key.Publisher -notlike $PublisherLike) { continue }

            # Prefer the vendor's QuietUninstallString verbatim; it already carries
            # the correct silent switches for that installer technology.
            $useVerbatim = [bool]$key.QuietUninstallString
            $command = if ($useVerbatim) { $key.QuietUninstallString } else { $key.UninstallString }
            if (-not $command) { continue }

            $parsed = Get-FmaUninstallExeAndArgs $command
            if (-not $parsed) { Write-Host "Fleet: could not parse uninstall string: $command"; continue }
            $exe = $parsed.Exe
            $uargs = $parsed.Args
            $isMsi = $exe -match '(?i)msiexec'

            if ($isMsi) {
                $uargs = $uargs -replace '(?i)/i(\{)', '/x$1'
                if ($uargs -notmatch '(?i)/x') { $uargs = ("/x $uargs").Trim() }
                if ($uargs -notmatch '(?i)/qn') { $uargs = ("$uargs /qn").Trim() }
                if ($uargs -notmatch '(?i)/norestart') { $uargs = ("$uargs /norestart").Trim() }
            } elseif (-not $useVerbatim -and $FallbackArgs -ne '') {
                $uargs = ("$uargs $FallbackArgs").Trim()
            }

            if (-not $isMsi -and -not (Test-Path -LiteralPath $exe)) {
                Write-Host "Fleet: opposite-scope uninstaller missing on disk: $exe"
                continue
            }

            Write-Host "Fleet: removing opposite-scope copy '$($key.DisplayName)'"
            Write-Host "  Command: $exe"
            Write-Host "  Args: $uargs"
            try {
                $opts = @{ FilePath = $exe; PassThru = $true; Wait = $true; NoNewWindow = $true }
                if ($uargs -ne '') { $opts.ArgumentList = $uargs }
                $p = Start-Process @opts
                Write-Host "  Exit code: $($p.ExitCode)"
            } catch {
                Write-Host "  WARNING: failed to remove opposite-scope copy: $_"
            }
        }
    }
}

try {
    Remove-FmaOtherScopeCopies -TargetScope $fmaTargetScope -DisplayNameLike $fmaDisplayNameLike -PublisherLike $fmaPublisherLike -FallbackArgs $fmaFallbackUninstallArgs
} catch {
    Write-Host "Fleet: warning during opposite-scope cleanup: $_"
}

# ---- App install ----

# Install Vivaldi silently, machine-wide (Chromium-based browser).
# Fleet runs installs as SYSTEM, so --system-level is required to install for
# all users under %ProgramFiles% (and register under HKLM). Without it the
# installer lands in the SYSTEM profile and is invisible to the real user.
$process = Start-Process -FilePath $env:INSTALLER_PATH `
  -ArgumentList "--vivaldi-silent --do-not-launch-chrome --system-level" `
  -NoNewWindow -PassThru -Wait
Exit $process.ExitCode
