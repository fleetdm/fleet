# =============================================================================
# Restore BitLocker protection on an already-encrypted Windows host
#
# Non-destructive: never decrypts, never removes a protector, never reboots.
#
# Exit codes:
#   0 = protection is On (or was already On)
#   1 = could not safely restore; escalate (details on stdout)
#   2 = a restart is pending; deliberately did NOT call EnableKeyProtectors
# =============================================================================

#Requires -RunAsAdministrator

$ErrorActionPreference = 'Continue'

$BL_NAMESPACE = 'root\CIMV2\Security\MicrosoftVolumeEncryption'

function Get-Vol {
    Get-CimInstance -Namespace $BL_NAMESPACE -ClassName Win32_EncryptableVolume `
        -Filter "DriveLetter='$($env:SystemDrive)'"
}

# A null return from CIM means the lookup failed, which is NOT evidence about the volume's actual state, so it must
# stop the script rather than be read as "no protectors" or "not encrypted".
function Get-VolOrFail {
    $v = Get-Vol
    if (-not $v) {
        Write-Output "FAIL: CIM returned no Win32_EncryptableVolume for $env:SystemDrive."
        Write-Output "      Either the volume is not BitLocker-capable or the CIM query failed."
        Write-Output "      Not treating this as a statement about the volume's state. Escalate."
        exit 1
    }
    return $v
}

# Re-reads the volume on every call so state is never stale, and fails loudly if either the lookup or the method
# call itself produces nothing.
function Invoke-VolumeMethod {
    param(
        [Parameter(Mandatory = $true)][string] $MethodName,
        [hashtable] $Arguments
    )
    $v = Get-VolOrFail
    if ($Arguments) {
        $r = Invoke-CimMethod -InputObject $v -MethodName $MethodName -Arguments $Arguments
    } else {
        $r = Invoke-CimMethod -InputObject $v -MethodName $MethodName
    }
    if ($null -eq $r) {
        Write-Output "FAIL: $MethodName returned no result (CIM method call failed). Escalate."
        exit 1
    }
    return $r
}

# For read-only status calls, any non-zero ReturnValue means the answer is unusable. Acting on an unreadable state
# is the exact failure this script exists to avoid, so stop rather than guess.
function Invoke-VolumeQuery {
    param(
        [Parameter(Mandatory = $true)][string] $MethodName,
        [hashtable] $Arguments
    )
    if ($Arguments) {
        $r = Invoke-VolumeMethod -MethodName $MethodName -Arguments $Arguments
    } else {
        $r = Invoke-VolumeMethod -MethodName $MethodName
    }
    if ($r.ReturnValue -ne 0) {
        Write-Output ("FAIL: $MethodName returned 0x{0:X8}; volume state is unreadable. Escalate." -f $r.ReturnValue)
        exit 1
    }
    return $r
}

# TPM-family protector types: 1=TPM, 4=TPM+PIN, 5=TPM+StartupKey, 6=TPM+PIN+StartupKey. Any of these lets the
# volume unseal at boot without human input (PIN variants prompt for a PIN, which is by design and still not a
# recovery prompt).
$TPM_FAMILY = 1, 4, 5, 6

# Returned by ProtectKeyWithTPM when the protector is already present, which for this script's purposes means
# success, not failure.
$FVE_E_PROTECTOR_EXISTS = 0x80310031L

function Get-ProtectorIdCount {
    param([Parameter(Mandatory = $true)][int] $Type)
    $r = Invoke-VolumeQuery -MethodName 'GetKeyProtectors' -Arguments @{ KeyProtectorType = [uint32]$Type }
    return @($r.VolumeKeyProtectorID).Count
}

function Get-TpmProtectorCount {
    $n = 0
    foreach ($t in $TPM_FAMILY) { $n += Get-ProtectorIdCount -Type $t }
    return $n
}

$conv = (Invoke-VolumeQuery -MethodName 'GetConversionStatus').ConversionStatus   # 1 = FullyEncrypted
$prot = (Invoke-VolumeQuery -MethodName 'GetProtectionStatus').ProtectionStatus   # 0 = off, 1 = on
$tpmCount = Get-TpmProtectorCount
$npCount = Get-ProtectorIdCount -Type 3
# Protector type 0 means "every type"
$allCount = Get-ProtectorIdCount -Type 0

# Raw manage-bde is the only place the suspend reboot-count is exposed. The text is localized, so only the
# parenthesised digit is parsed.
$statusRaw = (manage-bde -status $env:SystemDrive 2>&1 | Out-String)
$rebootCount = if ($statusRaw -match '\(\s*(\d+)') { [int]$matches[1] } else { $null }

Write-Output "volume            : $env:SystemDrive"
Write-Output "conversionStatus  : $conv (1 = FullyEncrypted)"
Write-Output "protectionStatus  : $prot (0 = off, 1 = on)"
Write-Output "tpmProtectors     : $tpmCount"
Write-Output "recoveryPasswords : $npCount"
Write-Output "allProtectors     : $allCount (every protector type)"
Write-Output "suspendRebootCount: $(if ($null -ne $rebootCount) { $rebootCount } else { 'none (indefinite suspend, or protection is on)' })"

if ($conv -ne 1) {
    Write-Output "FAIL: volume is not FullyEncrypted (conversionStatus=$conv). Not a suspended-protection case; leaving it alone."
    exit 1
}

if ($prot -eq 1 -and $tpmCount -gt 0) {
    Write-Output "OK: protection already on with a TPM protector present. Nothing to do."
    exit 0
}

# A volume that is On but has no TPM protector is a latent lockout: it cannot unseal at boot. Fix the protector even though protection is nominally on.
if ($prot -eq 1 -and $tpmCount -eq 0) {
    Write-Output "WARN: protection is on but NO TPM protector exists. This host will prompt for the recovery key at next boot."
}

if ($allCount -eq 0) {
    Write-Output "FAIL: no key protectors of any type. Escalate; do not attempt automated repair."
    exit 1
}

# Step 1: ensure an auto-unlock protector exists BEFORE EnableKeyProtectors.
if ($tpmCount -eq 0) {
    Write-Output "step 1: no TPM-family protector -> adding a TPM protector"
    $r = Invoke-VolumeMethod -MethodName 'ProtectKeyWithTPM'
    if ($r.ReturnValue -eq $FVE_E_PROTECTOR_EXISTS) {
        # The desired state is already satisfied.
        Write-Output "step 1: a TPM protector already exists (added between the check and this call); continuing"
    } elseif ($r.ReturnValue -ne 0) {
        Write-Output ("FAIL: ProtectKeyWithTPM returned 0x{0:X8}." -f $r.ReturnValue)
        Write-Output "      0x80310061 = FVE_E_POLICY_STARTUP_PIN_REQUIRED (policy requires a startup PIN;"
        Write-Output "                   a TPM-only protector is not allowed, so a PIN must be enrolled instead)."
        Write-Output "      NOT calling EnableKeyProtectors: enabling protectors with no auto-unlock protector"
        Write-Output "      present would cause a recovery prompt at the next boot."
        exit 1
    } else {
        Write-Output ("step 1: TPM protector added: {0}" -f $r.VolumeKeyProtectorID)
    }
} else {
    Write-Output "step 1: TPM-family protector already present ($tpmCount)"
}

# Step 2: refuse to call EnableKeyProtectors while a restart is staged.
$pendCbs = Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending'
$pendWu = Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired'
if ($pendCbs -or $pendWu) {
    Write-Output "step 2: a restart is PENDING (CBS=$pendCbs WindowsUpdate=$pendWu). Not enabling protectors now."
    if ($null -ne $rebootCount -and $rebootCount -gt 0) {
        Write-Output "        This suspension carries a reboot count ($rebootCount), so the restart both applies the"
        Write-Output "        staged update AND restores protection on its own. No need to re-run this script,"
        Write-Output "        though re-running is harmless and will confirm the result."
    } else {
        Write-Output "        This suspension is INDEFINITE (no reboot count), so the restart will NOT restore"
        Write-Output "        protection by itself. Restart this host, then RE-RUN this script to enable protectors."
    }
    Write-Output "        Either way the TPM protector checked/added in step 1 is already in place, so the"
    Write-Output "        eventual EnableKeyProtectors call will have an auto-unlock protector."
    exit 2
}
Write-Output "step 2: no pending restart"

# Step 3: enable the key protectors.
if ($prot -ne 1) {
    Write-Output "step 3: calling EnableKeyProtectors to protect the key again"
    $r = Invoke-VolumeMethod -MethodName 'EnableKeyProtectors'
    if ($r.ReturnValue -ne 0) {
        Write-Output ("FAIL: EnableKeyProtectors returned 0x{0:X8}" -f $r.ReturnValue)
        exit 1
    }
} else {
    Write-Output "step 3: protection already on, no EnableKeyProtectors call needed"
}

Start-Sleep -Seconds 5
$protAfter = (Invoke-VolumeQuery -MethodName 'GetProtectionStatus').ProtectionStatus
$tpmAfter = Get-TpmProtectorCount
Write-Output "result: protectionStatus=$protAfter tpmProtectors=$tpmAfter"

if ($protAfter -eq 1 -and $tpmAfter -gt 0) {
    Write-Output "OK: protection restored on $env:SystemDrive with an auto-unlock protector present."
    exit 0
}
Write-Output "FAIL: protection is still not correctly restored. Escalate."
exit 1
