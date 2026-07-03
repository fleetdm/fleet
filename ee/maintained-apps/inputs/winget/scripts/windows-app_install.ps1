# Fleet Pattern A (MSIX): converge this app on the MSIX package
# (https://github.com/fleetdm/fleet/issues/48248).
#
# Fleet manages this app as an MSIX package, but the app also shipped (or still
# ships) as a Win32 installer (exe/MSI). A leftover Win32 copy at ANY scope keeps
# the scope-blind patch policy (osquery's "programs" table) red while the MSIX is
# current, and leaves a stale, unmanaged copy on disk. Unlike the dual-variant
# Win32 apps -- where only the opposite scope is swept -- every Win32 copy of an
# MSIX-managed app is legacy, so BOTH Win32 uninstall hives are swept: HKLM, and
# HKEY_USERS for per-user installs (Fleet runs as SYSTEM, where HKCU maps to
# SYSTEM's own hive, so per-user copies are found under HKEY_USERS).
#
# Guards:
# - DisplayName matching is exact, mirrors the detection query's names, and
#   requires a publisher match, so unrelated software can't match.
# - MSIX registrations are never touched: PackageFullName-style keys and entries
#   under \WindowsApps\ are skipped, so a re-run can't remove the package this
#   script just installed.
# - Entries with no quiet uninstall path are skipped: a raw UninstallString run
#   as SYSTEM can hang on UI until Fleet's script timeout kills the install.
# - A per-user uninstaller launched by SYSTEM removes its files but cannot clean
#   the user's HKEY_USERS uninstall key (it writes to HKCU, which is SYSTEM's
#   hive). That phantom entry would keep the policy red forever, so the matched
#   key is deleted -- but ONLY after verifying the uninstaller removed itself
#   from disk. If files remain, the key stays and the policy stays truthfully red.
# - Best-effort: cleanup never aborts the MSIX install below.
#
# Data note: Win32 -> MSIX is a cross-packaging move; local app data does not
# carry into the MSIX container (account-based/server-synced data survives).

$fmaWin32Matchers = @(
    @{ Name = 'Windows App'; Publisher = 'Microsoft*'; FallbackArgs = '' }
)

function Get-FmaUninstallExeAndArgs {
    param([string]$Command)
    # Registry uninstall strings come in three shapes; parse defensively.
    if ($Command -match '^\s*"([^"]+)"\s*(.*)$') { return @{ Exe = $Matches[1]; Args = $Matches[2].Trim() } }
    elseif ($Command -match '(?i)^\s*(.+?\.exe)\s*(.*)$') { return @{ Exe = $Matches[1]; Args = $Matches[2].Trim() } }
    elseif ($Command -match '^\s*(\S+)\s*(.*)$') { return @{ Exe = $Matches[1]; Args = $Matches[2].Trim() } }
    return $null
}

function Remove-FmaWin32Copies {
    param([array]$Matchers)

    # Every Win32 scope is legacy for an MSIX-managed app: sweep HKLM and all
    # real interactive users' hives (skip .DEFAULT, service SIDs, _Classes).
    $roots = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    )
    foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
        if ($hive.Name -match '_Classes$') { continue }
        if ($hive.PSChildName -notmatch '^S-1-5-21-') { continue }
        $roots += "Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall"
        $roots += "Registry::$($hive.Name)\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall"
    }

    foreach ($root in $roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            # MSIX self-guard: a PackageFullName-style key name means an MSIX
            # registration, never a legacy Win32 copy.
            if ($sub.PSChildName -match '_[a-z0-9]{13}$') { continue }
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if (-not $key.DisplayName) { continue }
            if ("$($key.UninstallString)$($key.InstallLocation)" -match '(?i)\\WindowsApps\\') { continue }

            $matcher = $null
            foreach ($m in $Matchers) {
                if ($key.DisplayName -like $m.Name -and ($m.Publisher -eq '' -or $key.Publisher -like $m.Publisher)) {
                    $matcher = $m
                    break
                }
            }
            if (-not $matcher) { continue }

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
            } elseif (-not $useVerbatim) {
                if ($matcher.FallbackArgs -eq '') {
                    Write-Host "Fleet: no quiet uninstall path for legacy copy '$($key.DisplayName)', leaving it (policy stays red)"
                    continue
                }
                $uargs = ("$uargs $($matcher.FallbackArgs)").Trim()
            }

            if (-not $isMsi -and -not (Test-Path -LiteralPath $exe)) {
                Write-Host "Fleet: legacy Win32 uninstaller missing on disk: $exe"
                continue
            }

            Write-Host "Fleet: removing legacy Win32 copy '$($key.DisplayName)' from $root"
            Write-Host "  Command: $exe"
            Write-Host "  Args: $uargs"
            try {
                $opts = @{ FilePath = $exe; PassThru = $true; Wait = $true; NoNewWindow = $true }
                if ($uargs -ne '') { $opts.ArgumentList = $uargs }
                $p = Start-Process @opts
                Write-Host "  Exit code: $($p.ExitCode)"
            } catch {
                Write-Host "  WARNING: failed to remove legacy Win32 copy: $_"
                continue
            }

            # Per-user uninstallers can't clean their HKEY_USERS key when run as
            # SYSTEM. Remove the matched key only after the uninstaller is
            # verifiably gone from disk (Squirrel-style uninstallers self-delete
            # with a short delay, hence the settle time).
            if ($root -like 'Registry::HKEY_USERS*' -and -not $isMsi) {
                Start-Sleep -Seconds 5
                if (-not (Test-Path -LiteralPath $exe)) {
                    Remove-Item -Path $sub.PSPath -Recurse -Force -ErrorAction SilentlyContinue
                    Write-Host "  Removed leftover per-user uninstall registry entry"
                } else {
                    Write-Host "  Uninstaller still on disk; leaving registry entry (policy stays red)"
                }
            }
        }
    }
}

try {
    Remove-FmaWin32Copies -Matchers $fmaWin32Matchers
} catch {
    Write-Host "Fleet: warning during legacy Win32 cleanup: $_"
}

# ---- MSIX install ----

# MSIX: provision machine-wide so the app is available to all users at sign-in, then
# opportunistically register for the currently logged-on console user (via a scheduled
# task in their session) so the app is immediately visible without requiring sign-out.
#
# The Fleet agent runs as Local System on Windows, and Add-AppxPackage cannot run in that
# context (HRESULT 0x80073CF9). The scheduled task is the supported way to register a
# package in a user session from a system-context script.

$softwareName = "WindowsApp"
$taskName = "fleet-install-$softwareName.msix"
$scriptPath = "$env:PUBLIC\install-$softwareName.ps1"
$exitCodeFile = "$env:PUBLIC\install-exitcode-$softwareName.txt"

try {

  $msixPath = $env:INSTALLER_PATH
  if (-not $msixPath) {
    throw "INSTALLER_PATH is not set"
  }

  Write-Host "Provisioning MSIX for all users..."
  $result = Add-AppxProvisionedPackage -Online -PackagePath $msixPath -SkipLicense -ErrorAction Stop
  $result | Out-String | Write-Host

  # Win32_ComputerSystem.UserName returns the console user (DOMAIN\User) or null when no
  # interactive session is active. Other RDP/fast-user-switch sessions won't get the
  # immediate registration; those users will pick it up from the provisioned install at
  # their next sign-in.
  $userName = (Get-CimInstance Win32_ComputerSystem).UserName
  if (-not $userName -or $userName -notlike "*\*") {
    Write-Host "No interactive user logged on; provisioned install will register for each user at sign-in."
    Start-Sleep -Seconds 5
    Exit 0
  }

  Write-Host "Registering MSIX for logged-on user '$userName' via scheduled task..."

  $userScript = @"
`$msixPath = "$msixPath"
`$exitCodeFile = "$exitCodeFile"
try {
  Add-AppxPackage -Path `$msixPath -ErrorAction Stop | Out-String | Write-Host
  Set-Content -Path `$exitCodeFile -Value 0
} catch {
  Write-Host "Add-AppxPackage failed: `$(`$_.Exception.Message)"
  Set-Content -Path `$exitCodeFile -Value 1
}
"@

  Set-Content -Path $scriptPath -Value $userScript -Force

  $action = New-ScheduledTaskAction -Execute "powershell.exe" `
    -Argument "-WindowStyle Hidden -ExecutionPolicy Bypass -File `"$scriptPath`""
  $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries
  $principal = New-ScheduledTaskPrincipal -UserId $userName -RunLevel Highest
  $task = New-ScheduledTask -Action $action -Settings $settings -Principal $principal
  Register-ScheduledTask -TaskName $taskName -InputObject $task -User $userName -Force | Out-Null
  Start-ScheduledTask -TaskName $taskName

  $startDate = Get-Date
  $state = (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue).State
  while ($state -ne "Running") {
    Start-Sleep -Seconds 1
    if ((New-Timespan -Start $startDate).TotalSeconds -gt 30) {
      Write-Host "Per-user registration task did not start within 30s; provisioned install is still valid."
      break
    }
    $state = (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue).State
  }

  while ($state -eq "Running") {
    Start-Sleep -Seconds 2
    if ((New-Timespan -Start $startDate).TotalSeconds -gt 90) {
      Write-Host "Per-user registration task did not complete within 90s; provisioned install is still valid."
      break
    }
    $state = (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue).State
  }

  if (Test-Path $exitCodeFile) {
    $code = (Get-Content $exitCodeFile -ErrorAction SilentlyContinue | Select-Object -First 1).Trim()
    if ($code -eq "0") {
      Write-Host "Per-user registration completed for '$userName'."
    } else {
      Write-Host "Per-user registration did not complete cleanly (exit code: $code). Provisioned install is still valid."
    }
  }

  Start-Sleep -Seconds 5
  Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
} finally {
  Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
  Remove-Item -Path $scriptPath -Force -ErrorAction SilentlyContinue
  Remove-Item -Path $exitCodeFile -Force -ErrorAction SilentlyContinue
}
