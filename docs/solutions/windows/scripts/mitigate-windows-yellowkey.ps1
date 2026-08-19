<#
.SYNOPSIS
    Mitigates the YellowKey BitLocker bypass (CVE-2026-45585) by removing
    autofstx.exe from the WinRE image's Session Manager BootExecute value.

.DESCRIPTION
    Applies Microsoft's mitigation for YellowKey (CVE-2026-45585): strip
    autofstx.exe from the WinRE image's Session Manager BootExecute.
    Affects Windows 11, Server 2022, Server 2025. Safe to run on every
    affected host; runs unconditionally (no opt-in gate).

    The flow, from the CVE-2026-45585 MSRC advisory FAQ:
      1. reagentc /mountre to mount the WinRE image
      2. reg load the offline SYSTEM hive
      3. Walk every ControlSet and strip autofstx (any form: `autofstx`,
         `autofstx.exe`, `autocheck autofstx`, `autofstx.exe /flag`),
         verifying each via read-back
      4. reg unload, reagentc /unmountre /commit
      5. reagentc /disable + /enable to re-seal the BitLocker measurement chain

    Mount, hive, edit, and unmount run in one try/finally so the hive and
    mount are always released. The mount directory is under %SystemRoot%\Temp,
    ACL-locked to Administrators. On success the script writes
    HKLM\SOFTWARE\Fleet\YellowKey\BootExecMitigated = 1, which the
    windows_yellowkey extension reads to report the host as `mitigated`.

    One-way: no unmitigate. If a patch ships, apply it and clear the marker.
    TPM + PIN raises attacker cost but does not block the withheld variant.

.PARAMETER MountPath
    Directory to use as the WinRE mount point. Created if missing and ACL-
    locked to Administrators. Default: %SystemRoot%\Temp\fleet-yk-winre-mount

.OUTPUTS
    Structured key:value output to stdout for log capture.

.NOTES
    Exit codes:
      0 = autofstx removed, already absent, or WinRE already disabled
      3 = OS not affected (Windows 10 etc.); no action taken
      4 = Mount, edit, unmount, or re-seal failed; manual investigation needed

    References:
      MSRC CVE-2026-45585 (FAQ section contains the canonical Microsoft script):
        https://msrc.microsoft.com/update-guide/vulnerability/CVE-2026-45585
      Eclypsium technical analysis:
        https://eclypsium.com/blog/yellowkey-bitlocker-bypass-windows-recovery-environment/
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $false)]
    [string]$MountPath = (Join-Path $env:SystemRoot 'Temp\fleet-yk-winre-mount')
)

$ErrorActionPreference = 'Stop'

function Write-State {
    param([string]$Label, [string]$Value)
    Write-Output ("{0,-30} : {1}" -f $Label, $Value)
}

function Lock-AdminOnlyAcl {
    # Lock the mount directory to Administrators-only access. Defends against
    # TOCTOU between empty-check and reagentc /mountre, and against non-admin
    # local DoS where a user pre-populates the directory to trip mount_dir_dirty.
    param([string]$Path)
    try {
        $acl = Get-Acl $Path
        $acl.SetAccessRuleProtection($true, $false)
        $rule = New-Object System.Security.AccessControl.FileSystemAccessRule(
            'BUILTIN\Administrators','FullControl',
            'ContainerInherit,ObjectInherit','None','Allow')
        $acl.AddAccessRule($rule)
        Set-Acl $Path $acl
    } catch {
        Write-Output "WARN: could not lock ACL on $Path : $($_.Exception.Message)"
    }
}

Write-Output "=== Windows YellowKey mitigation (autofstx strip) ==="
Write-Output ""

# Match every reasonable spelling of the entry: 'autofstx', 'autofstx.exe',
# 'autocheck autofstx', 'autofstx.exe /flag'. Word-boundary anchored,
# case-insensitive.
$AutofstxPattern = '(?i)\bautofstx(\.exe)?\b'
$HiveName        = 'YK_WinREHive'

$hiveLoaded   = $false
$imageMounted = $false
$mountCreated = $false
$changesMade  = $false
$editClean    = $false

# --- Fleet: success marker path (BootExecMitigated is written on success;
#     no opt-in gate because Microsoft's autofstx strip is the official
#     mitigation and is safe to apply on every affected host). ---
$markerPath = 'HKLM:\SOFTWARE\Fleet\YellowKey'

# --- Fleet: OS check ---
$os = (Get-CimInstance Win32_OperatingSystem).Caption
Write-State "OS" $os
$affected = ($os -match 'Windows 11' -or $os -match 'Server 2022' -or $os -match 'Server 2025')
if (-not $affected) {
    Write-Output "SKIP: $os is not in YellowKey's affected OS list."
    Write-State "State" "skipped_os_not_affected"
    exit 3
}

# --- Admin check ---
try {
    $identity  = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]$identity
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Output "FAIL: must run as Administrator."
        Write-State "State" "not_admin"
        exit 4
    }
} catch {
    Write-Output "FAIL: admin check error: $($_.Exception.Message)"
    Write-State "State" "admin_check_failed"
    exit 4
}

# --- WinRE state (CJK-colon tolerant; localized values fall through to error) ---
$winreOutput = & reagentc /info 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Output "FAIL: reagentc /info exit $LASTEXITCODE"
    Write-State "State" "reagentc_info_failed"
    exit 4
}
$winreText = $winreOutput -join "`n"
if ($winreText -match "[:：]\s*Disabled\b") {
    Write-Output "OK: WinRE disabled. Stronger mitigation already in place; nothing to do."
    Write-State "State" "winre_already_disabled"
    exit 0
}
if ($winreText -notmatch "[:：]\s*Enabled\b") {
    Write-Output "FAIL: could not parse reagentc /info output (locale not supported by current regex)."
    Write-State "State" "winre_state_unknown"
    exit 4
}
Write-State "WinRE status" "Enabled"

# --- All mount/load/edit/unload/unmount work inside one try/finally so the
#     hive handle and the mounted image are always released, even on a
#     thrown exception mid-flow. exit code is decided after finally based
#     on $editClean. ---
try {
    # --- Prepare mount directory (admin-only path, ACL-locked) ---
    if (-not (Test-Path $MountPath)) {
        New-Item -ItemType Directory -Path $MountPath -Force | Out-Null
        $mountCreated = $true
        Lock-AdminOnlyAcl -Path $MountPath
    } else {
        $existing = Get-ChildItem -Path $MountPath -Force -ErrorAction SilentlyContinue
        if ($existing) {
            Write-Output "FAIL: $MountPath not empty. Clean it or pass -MountPath."
            Write-State "State" "mount_dir_dirty"
            throw "mount_dir_dirty"
        }
        # Lock the ACL even on a pre-existing directory to defend against
        # a non-admin user creating it earlier.
        Lock-AdminOnlyAcl -Path $MountPath
    }

    # --- Mount WinRE image ---
    $mountOutput = & reagentc /mountre /path $MountPath 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Output "FAIL: reagentc /mountre: $mountOutput"
        Write-State "State" "mount_failed"
        throw "mount_failed"
    }
    $imageMounted = $true
    Write-State "Mounted at" $MountPath

    # --- Locate offline SYSTEM hive ---
    $hivePath = $null
    foreach ($candidate in @(
        "$MountPath\Windows\System32\config\SYSTEM",
        "$MountPath\windows\system32\config\SYSTEM"
    )) {
        if (Test-Path $candidate) { $hivePath = $candidate; break }
    }
    if (-not $hivePath) {
        $found = Get-ChildItem -Path $MountPath -Recurse -Filter 'SYSTEM' -ErrorAction SilentlyContinue |
                 Where-Object { $_.FullName -match 'config\\SYSTEM$' } | Select-Object -First 1
        if ($found) { $hivePath = $found.FullName }
    }
    if (-not $hivePath) {
        Write-Output "FAIL: SYSTEM hive not found in mounted image."
        Write-State "State" "hive_not_found"
        throw "hive_not_found"
    }

    # --- Load offline SYSTEM hive ---
    & reg load "HKLM\$HiveName" $hivePath 2>$null | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Output "FAIL: reg load exit $LASTEXITCODE"
        Write-State "State" "reg_load_failed"
        throw "reg_load_failed"
    }
    $hiveLoaded = $true
    Write-State "Hive loaded" "HKLM\$HiveName"

    # --- Enumerate every ControlSet child key directly. Covers Current,
    #     Default, LastKnownGood, Failed, and any rolled-back snapshots. ---
    $controlSets = @()
    try {
        $controlSets = @(
            Get-ChildItem "Registry::HKEY_LOCAL_MACHINE\$HiveName" -ErrorAction Stop |
            Where-Object { $_.PSChildName -like 'ControlSet*' } |
            ForEach-Object { $_.PSChildName }
        )
    } catch {
        $controlSets = @()
    }
    if ($controlSets.Count -eq 0) {
        $controlSets = @('ControlSet001')
    }
    Write-State "ControlSets" ($controlSets -join ', ')

    # --- Strip autofstx from each ControlSet's BootExecute, verify read-back ---
    foreach ($cs in $controlSets) {
        $regPath = "Registry::HKEY_LOCAL_MACHINE\$HiveName\$cs\Control\Session Manager"
        $cur = (Get-ItemProperty -Path $regPath -Name 'BootExecute' -ErrorAction SilentlyContinue).BootExecute
        if (-not $cur) {
            Write-State "$cs" "no_bootexecute"
            continue
        }
        $curArr = @($cur)
        $newArr = @($curArr | Where-Object { $_ -and ($_ -notmatch $AutofstxPattern) })
        if ($newArr.Count -eq $curArr.Count) {
            Write-State "$cs" "autofstx absent"
            continue
        }
        Set-ItemProperty -Path $regPath -Name 'BootExecute' -Value $newArr -Type MultiString

        # Verify read-back. Refuse to claim success if the strip did not stick.
        $verify = (Get-ItemProperty -Path $regPath -Name 'BootExecute' -ErrorAction Stop).BootExecute
        if (@($verify) | Where-Object { $_ -match $AutofstxPattern }) {
            Write-State "$cs" "verify_failed_autofstx_still_present"
            throw "verify_failed_$cs"
        }
        $changesMade = $true
        Write-State "$cs" "stripped autofstx"
    }

    # The edit loop finished without throw. Set the clean flag last so any
    # exception above leaves it false.
    $editClean = $true
}
catch {
    # State has already been written inside the try block before each throw.
    # If a cmdlet inside the edit loop raised an unexpected error (no prior
    # State write), surface it as edit_error.
    if ($_.Exception.Message -notmatch '^(mount_dir_dirty|mount_failed|hive_not_found|reg_load_failed|verify_failed_)') {
        Write-Output "FAIL: $($_.Exception.Message)"
        Write-State "State" "edit_error"
    }
}
finally {
    # Always release the hive and unmount, regardless of how we got here.
    if ($hiveLoaded) {
        [gc]::Collect()
        [gc]::WaitForPendingFinalizers()
        Start-Sleep -Seconds 2
        & reg unload "HKLM\$HiveName" 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            [gc]::Collect()
            Start-Sleep -Seconds 3
            & reg unload "HKLM\$HiveName" 2>$null | Out-Null
        }
    }
    if ($imageMounted) {
        $flag = if ($editClean -and $changesMade) { '/commit' } else { '/discard' }
        & reagentc /unmountre /path $MountPath $flag 2>$null | Out-Null
    }
    if ($mountCreated -and (Test-Path $MountPath)) {
        Remove-Item -Path $MountPath -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# --- If anything in the try block threw, bail without writing the marker ---
if (-not $editClean) {
    exit 4
}

# --- Re-seal BitLocker measurement chain (only when changes were committed) ---
# disable and enable are checked independently because $LASTEXITCODE is
# overwritten by each external command.
if ($changesMade) {
    & reagentc /disable 2>$null | Out-Null
    $disableExit = $LASTEXITCODE
    & reagentc /enable 2>$null | Out-Null
    $enableExit = $LASTEXITCODE
    if ($disableExit -ne 0 -or $enableExit -ne 0) {
        Write-Output "FAIL: reseal failed (disable=$disableExit, enable=$enableExit). Run reagentc /enable manually if needed."
        Write-State "State" "reseal_failed"
        exit 4
    }
    Write-State "WinRE re-sealed" "disable + enable"
}

# --- Fleet: success marker. Only written when the edit loop completed
#     cleanly AND every ControlSet read-back verified autofstx absent. ---
try {
    if (-not (Test-Path $markerPath)) {
        New-Item -Path $markerPath -Force | Out-Null
    }
    Set-ItemProperty -Path $markerPath -Name 'BootExecMitigated' -Value 1 -Type DWord -Force
} catch {
    Write-Output "WARN: could not write BootExecMitigated marker: $($_.Exception.Message)"
}

if ($changesMade) {
    Write-State "State" "bootexec_stripped"
} else {
    Write-State "State" "bootexec_already_stripped"
}
exit 0
