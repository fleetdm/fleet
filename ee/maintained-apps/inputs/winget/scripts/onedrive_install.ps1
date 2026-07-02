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
$fmaDisplayNameLike       = "Microsoft OneDrive*"
$fmaPublisherLike         = ""
$fmaFallbackUninstallArgs = "/uninstall"

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

# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# OneDriveSetup.exe performs a per-machine install with "/allusers /silent"
# (switches verified against the winget InstallerSwitches Custom: /allusers and
# silentinstallhq.com). The catch: OneDriveSetup.exe spawns several child
# processes and starts the resident OneDrive.exe, so a plain Start-Process -Wait
# can wait indefinitely and hit the CI step timeout. Instead, start the
# installer, then poll for the per-machine install's registry uninstall entry
# and return success once it is registered.

$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/allusers /silent" -PassThru

# Per-machine OneDrive drops OneDrive.exe under Program Files and registers an
# uninstall entry -- which is what Fleet's detection (osquery "programs") reads.
# The binary lands BEFORE the entry is registered, so success requires the
# registry entry; exiting on the binary alone races detection. Modern x64
# builds register under the native hive, older ones under WOW6432Node.
# OneDriveSetup.exe may also exit while a child process finishes the
# registration, so keep polling until the deadline even after it exits.
$uninstallKeys = @(
  "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\OneDriveSetup.exe",
  "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\OneDriveSetup.exe"
)
$uninstallRoots = @(
  "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
  "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall"
)
$exePaths = @(
  "$env:ProgramFiles\Microsoft OneDrive\OneDrive.exe",
  "${env:ProgramFiles(x86)}\Microsoft OneDrive\OneDrive.exe"
)

function Test-OneDriveRegistered {
  foreach ($k in $uninstallKeys) {
    if (Test-Path $k) { return $true }
  }
  # Fallback in case the key name drifts across OneDrive builds.
  $entry = Get-ChildItem -Path $uninstallRoots -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
    Where-Object { $_.DisplayName -eq 'Microsoft OneDrive' } |
    Select-Object -First 1
  return [bool]$entry
}

$timeoutSeconds = 240
$deadline = (Get-Date).AddSeconds($timeoutSeconds)
$registered = $false

while ((Get-Date) -lt $deadline) {
  if (Test-OneDriveRegistered) { $registered = $true; break }
  Start-Sleep -Seconds 5
}

if ($registered) {
  Write-Host "OneDrive per-machine install registered."
  Exit 0
}

$exeExists = $false
foreach ($p in $exePaths) {
  if ($p -and (Test-Path $p)) { $exeExists = $true; break }
}
if ($exeExists) {
  Write-Host "Warning: OneDrive.exe present but uninstall entry not registered within $timeoutSeconds seconds."
  Exit 0
}

if ($process.HasExited) {
  $exitCode = $process.ExitCode
  Write-Host "OneDriveSetup exited with code: $exitCode"
  if ($exitCode -eq 0 -or $exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
  Exit $exitCode
}

Write-Host "Timed out waiting for OneDrive install to complete."
Exit 1

} catch {
  Write-Host "Error: $_"
  Exit 1
}
