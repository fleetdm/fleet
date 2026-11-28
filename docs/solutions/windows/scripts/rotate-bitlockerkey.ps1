# Please don't delete. This script is referenced in the guide here:
# https://fleetdm.com/guides/rotate-bitlocker-recovery-keys-with-a-script
<#
.SYNOPSIS
    Rotates the BitLocker recovery password on the OS volume and escrows the new
    one to Fleet through the same agent endpoint fleetd uses.

.DESCRIPTION
    This calls an internal fleetd endpoint:
        POST /api/fleet/orbit/disk_encryption_key
        { "orbit_node_key": "...", "encryption_key": "<base64>", "client_error": "" }
    Fleet answers 204 No Content on success and logs an
    "escrowed disk encryption key" activity on the host.

    Written against fleetdm/fleet main as of 2026-11-27 (Fleet 4.95.x).
    Re-check client/orbit_client.go and server/service/orbit.go after every upgrade.

    Order of operations, so the drive always has a recovery key Fleet can show:
      0. Confirm Fleet's disk encryption status for the host is "verified", through
         the device endpoint Fleet Desktop uses (GET /api/latest/fleet/device/{token}).
         Fleet skips storing keys when disk encryption is off but still answers 204.
      1. Add a new recovery password protector.
      2. Escrow it to Fleet.
      3a. Fleet confirmed (204)        -> remove the old recovery passwords.
      3b. Fleet rejected it (HTTP 4xx) -> remove the NEW protector (Fleet's key stays valid).
      3c. Anything else (no response,  -> keep BOTH protectors; both keys still unlock the drive.
          3xx, 5xx, other 2xx)            A 5xx can come from a proxy after Fleet stored the key.

    Requests are HTTPS only, never follow redirects, and allow TLS 1.2 or newer.

    Every step is logged to standard output so it appears in Fleet's script output.
    The fleetd node key is never logged, and only the first and last groups of the
    recovery password are logged (e.g. 123456-******-...-******-654321).
    Exit code 0 = rotated, escrowed, and every old recovery password removed;
    1 = failed or incomplete (see the log for the state left behind).
#>
#Requires -RunAsAdministrator
$ErrorActionPreference = 'Stop'

# ------------------------------- configuration -------------------------------
# Leave $FleetUrl and $OrbitRoot empty to detect them from the fleetd service
# ("Fleet osquery") registry key. Set either one to override detection.
$FleetUrl   = ''                                   # e.g. 'https://fleet.example.com'
$OrbitRoot  = ''                                   # e.g. 'C:\Program Files\Orbit'
$MountPoint = $env:SystemDrive                     # usually C:
# -----------------------------------------------------------------------------

$FleetdServiceKey = 'HKLM:\SYSTEM\CurrentControlSet\Services\Fleet osquery'

function Write-Log {
    param([string]$Message, [string]$Level = 'INFO')
    Write-Output ("{0} [{1}] {2}" -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $Level, $Message)
}

# Reads a setting from the fleetd service. Current installers put settings in the
# service's Environment value (e.g. ORBIT_FLEET_URL=...); older ones passed them as
# command-line flags in ImagePath (e.g. --fleet-url=...), so check both.
function Get-FleetdSetting {
    param([string]$EnvName, [string]$FlagName)
    $svc = Get-ItemProperty -Path $FleetdServiceKey -ErrorAction SilentlyContinue
    if (-not $svc) { return $null }
    foreach ($line in @($svc.Environment)) {
        if ($line -match "^$EnvName=(.+)$") { return @{ Value = $Matches[1].Trim(); Source = "service Environment ($EnvName)" } }
    }
    # Quoted values can contain spaces (e.g. --root-dir="C:\Program Files\Orbit").
    if ($svc.ImagePath -match "--$FlagName(?:=|\s+)(?:`"([^`"]*)`"|([^`"\s]+))") {
        # Only one of the two groups matches; joining them avoids a $null for an empty quoted value.
        return @{ Value = "$($Matches[1])$($Matches[2])".Trim(); Source = "service ImagePath (--$FlagName)" }
    }
    return $null
}

function Get-RecoveryProtectors {
    @((Get-BitLockerVolume -MountPoint $MountPoint).KeyProtector |
        Where-Object KeyProtectorType -eq 'RecoveryPassword')
}

function Write-ProtectorState {
    param([string]$Label)
    $all = (Get-BitLockerVolume -MountPoint $MountPoint).KeyProtector
    Write-Log "$Label - $(@($all).Count) key protector(s) on ${MountPoint}"
    foreach ($p in $all) { Write-Log "    $($p.KeyProtectorType)  $($p.KeyProtectorId)" }
}

$exitCode = 1
try {
    Write-Log "BitLocker recovery key rotation starting on $env:COMPUTERNAME"

    # ------------------------- detect fleetd settings ------------------------
    if ($FleetUrl) {
        Write-Log "Fleet URL: $FleetUrl (set in script)"
    }
    else {
        $found = Get-FleetdSetting -EnvName 'ORBIT_FLEET_URL' -FlagName 'fleet-url'
        if (-not $found -or -not $found.Value) {
            throw "Could not detect the Fleet URL from '$FleetdServiceKey'. Set `$FleetUrl at the top of the script. No changes made."
        }
        $FleetUrl = $found.Value
        Write-Log "Fleet URL: $FleetUrl (detected from $($found.Source))"
    }
    $FleetUrl = $FleetUrl.TrimEnd('/')
    if ($FleetUrl -notmatch '^https://') {
        throw "Fleet URL '$FleetUrl' isn't an https:// URL. The script won't send the recovery key over plain HTTP. No changes made."
    }

    if ($OrbitRoot) {
        Write-Log "fleetd root: $OrbitRoot (set in script)"
    }
    else {
        $found = Get-FleetdSetting -EnvName 'ORBIT_ROOT_DIR' -FlagName 'root-dir'
        if ($found -and $found.Value) {
            $OrbitRoot = [IO.Path]::GetFullPath($found.Value)   # installer writes e.g. 'C:\Program Files\Orbit\.'
            Write-Log "fleetd root: $OrbitRoot (detected from $($found.Source))"
        }
        else {
            $OrbitRoot = 'C:\Program Files\Orbit'
            Write-Log "fleetd root: $OrbitRoot (not found in registry; using default)"
        }
    }
    if ((Get-FleetdSetting -EnvName 'ORBIT_FLEET_CERTIFICATE' -FlagName 'fleet-certificate')) {
        Write-Log "fleetd uses a custom Fleet server certificate; this request relies on Windows trusting that CA" 'WARN'
    }
    Write-Log "Volume: $MountPoint"

    $NodeKeyPath     = Join-Path $OrbitRoot 'secret-orbit-node-key.txt'
    $DeviceTokenPath = Join-Path $OrbitRoot 'identifier'
    $EscrowUri       = "$FleetUrl/api/fleet/orbit/disk_encryption_key"

    # ----------------------------- preflight ---------------------------------
    Write-Log "Step 0: preflight checks"
    $vol = Get-BitLockerVolume -MountPoint $MountPoint
    Write-Log "Volume status: $($vol.VolumeStatus) | Protection: $($vol.ProtectionStatus) | Method: $($vol.EncryptionMethod)"
    if ($vol.VolumeStatus -ne 'FullyEncrypted') {
        throw "$MountPoint is '$($vol.VolumeStatus)'. Only rotate a fully encrypted volume. No changes made."
    }
    if (-not (Test-Path $NodeKeyPath)) {
        throw "fleetd node key not found at $NodeKeyPath. No changes made."
    }
    Write-Log "fleetd node key file found"
    if (Test-Path (Join-Path $OrbitRoot 'host_identity.crt')) {
        throw "Host uses a Fleet host identity certificate; unsigned escrow requests would be rejected. No changes made."
    }
    Write-Log "No host identity certificate present"
    $nodeKey = (Get-Content -Raw -Path $NodeKeyPath).Trim()
    # Add TLS 1.2 to the allowed protocols without removing newer ones (e.g. TLS 1.3).
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    # The escrow endpoint answers 204 without storing anything when disk encryption
    # is off for the host's fleet, and it can't confirm which key it stored. Fleet
    # reports a BitLocker status only when disk encryption is on, so require
    # "verified" (Fleet holds a working key) before changing anything.
    if (-not (Test-Path $DeviceTokenPath)) {
        throw "fleetd device token not found at $DeviceTokenPath. No changes made."
    }
    $deviceToken = (Get-Content -Raw -Path $DeviceTokenPath).Trim()
    Write-Log "Checking the host's disk encryption status in Fleet"
    $deStatus = $null
    try {
        # The token is part of the URL, so never log the URL or the raw exception.
        $resp = Invoke-WebRequest -Uri "$FleetUrl/api/latest/fleet/device/$deviceToken`?exclude_software=true" `
            -Method Get -UseBasicParsing -TimeoutSec 60 -MaximumRedirection 0
        $deviceStatus = [int]$resp.StatusCode
        if ($deviceStatus -eq 200) {
            $deStatus = ($resp.Content | ConvertFrom-Json).host.mdm.os_settings.disk_encryption.status
        }
    }
    catch {
        $deviceStatus = if ($_.Exception.Response) { [int]$_.Exception.Response.StatusCode } else { $null }
    }
    finally {
        $deviceToken = $null
        $resp = $null
    }
    if ($deviceStatus -ne 200) {
        throw "Could not read the host's disk encryption status from Fleet ($(if ($deviceStatus) { "HTTP $deviceStatus" } else { 'no response' })). No changes made."
    }
    if (-not $deStatus) {
        throw "Fleet reports no disk encryption status for this host, so it wouldn't store a new key. Turn on disk encryption for the host's fleet. No changes made."
    }
    if ($deStatus -ne 'verified') {
        throw "Fleet's disk encryption status for this host is '$deStatus'. Only rotate when it's 'verified'. No changes made."
    }
    Write-Log "Fleet disk encryption status: verified"
    Write-ProtectorState "Before rotation"

    # -------------------------- 1. add new protector -------------------------
    Write-Log "Step 1: adding new recovery password protector"
    $oldIds = @(Get-RecoveryProtectors | ForEach-Object KeyProtectorId)
    Write-Log "Existing recovery password protector(s): $(if ($oldIds) { $oldIds -join ', ' } else { 'none' })"

    # This cmdlet prints the full recovery password as a warning ("ACTIONS REQUIRED"),
    # which Fleet would capture in the script output. Suppress the warning stream.
    Add-BitLockerKeyProtector -MountPoint $MountPoint -RecoveryPasswordProtector `
        -WarningAction SilentlyContinue 3>$null | Out-Null

    $newProtector = @(Get-RecoveryProtectors | Where-Object { $_.KeyProtectorId -notin $oldIds })
    if ($newProtector.Count -ne 1) {
        Write-ProtectorState "Unexpected state"
        throw "Expected exactly one new recovery protector, found $($newProtector.Count). Nothing removed; review protectors above."
    }
    $newProtector = $newProtector[0]
    $newId = $newProtector.KeyProtectorId
    Write-Log "New recovery password protector added: $newId"
    # Only the first and last 6-digit groups are logged, enough to match the key Fleet shows.
    $groups    = $newProtector.RecoveryPassword -split '-'
    $keyForLog = $groups[0] + '-' + ('******-' * ($groups.Count - 2)) + $groups[-1]
    Write-Log "New recovery password (masked): $keyForLog"

    # --------------------------- 2. escrow to Fleet --------------------------
    Write-Log "Step 2: escrowing new key to Fleet ($EscrowUri)"
    $body = @{
        orbit_node_key = $nodeKey
        # Go marshals []byte as base64, so send base64 of the 48-digit password string.
        encryption_key = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($newProtector.RecoveryPassword))
        client_error   = ''
    } | ConvertTo-Json -Compress

    $outcome = 'unknown'
    $status  = $null
    try {
        # -MaximumRedirection 0: never resend the key and node key to another location.
        $resp   = Invoke-WebRequest -Uri $EscrowUri -Method Post -Body $body `
            -ContentType 'application/json' -UseBasicParsing -TimeoutSec 60 -MaximumRedirection 0
        $status = [int]$resp.StatusCode
        Write-Log "Fleet responded HTTP $status"
    }
    catch {
        if ($_.Exception.Response) {
            $status = [int]$_.Exception.Response.StatusCode
            Write-Log "Fleet responded HTTP $status : $($_.Exception.Message)" 'ERROR'
        }
        else {
            Write-Log "No response from Fleet: $($_.Exception.Message)" 'ERROR'
        }
    }
    finally {
        $body = $null
    }

    # Only a 204 is success, and only a 4xx is a definite rejection. Everything else
    # (no response, redirects, 5xx from Fleet or a proxy, other 2xx) is unknown,
    # because Fleet may have stored the key.
    if ($status -eq 204) {
        $outcome = 'escrowed'
    }
    elseif ($status -ge 400 -and $status -lt 500) {
        $outcome = 'rejected'
    }
    elseif ($status) {
        Write-Log "HTTP $status doesn't confirm whether Fleet stored the key; treating the outcome as unknown" 'WARN'
    }
    Write-Log "Escrow outcome: $outcome"

    # ------------------------------- 3. finish -------------------------------
    Write-Log "Step 3: finishing based on escrow outcome"
    switch ($outcome) {
        'escrowed' {
            $failedRemovals = 0
            foreach ($id in $oldIds) {
                try {
                    Remove-BitLockerKeyProtector -MountPoint $MountPoint -KeyProtectorId $id | Out-Null
                    Write-Log "Removed old recovery password protector $id"
                }
                catch {
                    $failedRemovals++
                    Write-Log "Could not remove old protector ${id}: $($_.Exception.Message)" 'WARN'
                }
            }
            Write-ProtectorState "After rotation"
            Write-Log "Key Fleet should now show (masked): $keyForLog"
            if ($failedRemovals -gt 0) {
                # The old key still unlocks the drive, so the shared key isn't invalidated yet.
                Write-Log "FAILED: new key escrowed, but $failedRemovals old protector(s) remain and still unlock the drive. New protector ID: $newId. Run the script again to remove them." 'ERROR'
            }
            else {
                Write-Log "SUCCESS: recovery key rotated and escrowed. New protector ID: $newId"
                $exitCode = 0
            }
        }
        'rejected' {
            Remove-BitLockerKeyProtector -MountPoint $MountPoint -KeyProtectorId $newId | Out-Null
            Write-Log "Removed new protector $newId; the key Fleet already holds is unchanged"
            Write-ProtectorState "After rollback"
            Write-Log "FAILED: Fleet rejected the escrow (HTTP $status)." 'ERROR'
        }
        default {
            Write-ProtectorState "Current state"
            Write-Log "FAILED: escrow outcome unknown$(if ($status) { " (HTTP $status)" }). Kept BOTH old and new protectors so whichever key Fleet holds still works. New protector ID: $newId. Re-run once the problem is fixed." 'ERROR'
        }
    }
}
catch {
    Write-Log "FAILED: $($_.Exception.Message)" 'ERROR'
    try { Write-ProtectorState "State at failure" } catch { }
}
finally {
    $nodeKey = $null
    $keyForLog = $null
    Write-Log "Finished with exit code $exitCode"
}
exit $exitCode
