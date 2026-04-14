# Please don't delete. This script is referenced in the guide here:
# https://fleetdm.com/guides/lock-wipe-hosts
#
# wipe-windows-device.ps1
# Fallback script to wipe a Windows device when the native MDM wipe command
# (doWipe/doWipeProtected) fails with status 500 or the device is not wiped.
#
# When Fleet sends a wipe via the RemoteWipe CSP, the command may fail due to:
# - Disabled or missing Windows Recovery Environment (WinRE)
# - Broken MDM enrollment state
# - Server-side command processing errors (DB timeouts, auth failures)
#
# This script bypasses the MDM command queue by calling the RemoteWipe CSP
# locally via the WMI-to-CSP bridge. Before triggering the wipe, it validates
# and repairs WinRE (the confirmed root cause of most failures) and suspends
# BitLocker to prevent recovery key prompts.
#
# Note: Every fully unattended Windows wipe method ultimately calls the same
# RemoteWipe CSP. There is no alternative Windows API for triggering "Reset
# this PC" programmatically without user interaction. The value of this script
# is that it fixes the root causes before calling the wipe, and bypasses the
# MDM command queue where server-side failures can occur.
#
# The OS is never formatted. Windows rebuilds from the local Component Store
# (WinSxS) or via Cloud Download, so no USB media is required.
#
# Usage: Run via Fleet on affected Windows hosts. Fully unattended.

#Requires -RunAsAdministrator

# Log output for audit trail
$logPath = "$env:ProgramData\fleet-wipe-device-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
Start-Transcript -Path $logPath -ErrorAction SilentlyContinue | Out-Null

$exitCode = 0

Write-Host "=== Fleet Windows Device Wipe (Fallback) ==="
Write-Host "Log file: $logPath"
Write-Host ""

# ---------------------------------------------------------------------------
# 1. Validate and repair WinRE
# ---------------------------------------------------------------------------
# The RemoteWipe CSP depends on WinRE to perform the reset. If WinRE is
# disabled or missing, the CSP returns status 500.
# Ref: https://github.com/fleetdm/fleet/issues/34994#issuecomment-2507872412
# ---------------------------------------------------------------------------
Write-Host "[1/4] Checking Windows Recovery Environment (WinRE)..."

$reagentInfo = reagentc /info 2>&1 | Out-String
if ($reagentInfo -match "Windows RE status:\s+Enabled") {
    Write-Host "  WinRE is enabled"
} else {
    Write-Host "  WinRE is disabled or missing - attempting to re-enable..."
    $enableResult = reagentc /enable 2>&1 | Out-String

    if ($LASTEXITCODE -eq 0) {
        # Verify WinRE was actually enabled by re-running reagentc /info
        $reagentInfoAfter = reagentc /info 2>&1 | Out-String
        if ($reagentInfoAfter -match "Windows RE status:\s+Enabled") {
            Write-Host "  WinRE re-enabled successfully"
        } else {
            Write-Host "  WARNING: reagentc /enable returned success but WinRE is not enabled"
            # Fall through to manual recovery attempt
            $enableResult = ""
        }
    }

    if ($LASTEXITCODE -ne 0 -or $enableResult -eq "") {
        $winreFound = $false
        $winreLocations = @(
            "$env:SystemDrive\Recovery\WindowsRE",
            "$env:SystemDrive\Windows\System32\Recovery"
        )

        foreach ($loc in $winreLocations) {
            if (Test-Path (Join-Path $loc "winre.wim")) {
                Write-Host "  Found winre.wim at $loc - registering..."
                reagentc /setreimage /path $loc 2>&1 | Out-Null
                $retryResult = reagentc /enable 2>&1 | Out-String
                if ($LASTEXITCODE -eq 0) {
                    # Verify WinRE was actually enabled
                    $reagentInfoRetry = reagentc /info 2>&1 | Out-String
                    if ($reagentInfoRetry -match "Windows RE status:\s+Enabled") {
                        Write-Host "  WinRE re-enabled using $loc"
                        $winreFound = $true
                        break
                    }
                }
            }
        }

        if (-not $winreFound) {
            Write-Host "  WARNING: Could not enable WinRE"
            Write-Host "  The wipe will likely fail without WinRE"
        }
    }
}
Write-Host ""

# ---------------------------------------------------------------------------
# 2. Check Component Store health
# ---------------------------------------------------------------------------
# The reset rebuilds Windows from the Component Store (WinSxS). If the store
# is corrupted, the reset may fail or produce a broken installation.
# ---------------------------------------------------------------------------
Write-Host "[2/4] Checking Component Store (WinSxS) integrity..."

$dismResult = & dism /Online /Cleanup-Image /ScanHealth /English 2>&1 | Out-String
if ($dismResult -match "No component store corruption detected") {
    Write-Host "  Component Store is healthy"
} elseif ($dismResult -match "The component store is repairable\.") {
    Write-Host "  Corruption detected - attempting repair..."
    $repairResult = & dism /Online /Cleanup-Image /RestoreHealth /English 2>&1 | Out-String
    if ($repairResult -match "completed successfully") {
        Write-Host "  Component Store repaired"
    } else {
        Write-Host "  WARNING: Repair failed - device may use Cloud Download as fallback"
    }
} else {
    Write-Host "  WARNING: Component Store state unexpected - continuing"
}
Write-Host ""

# ---------------------------------------------------------------------------
# 3. Suspend BitLocker
# ---------------------------------------------------------------------------
# Suspending BitLocker for one reboot cycle prevents the device from prompting
# for a recovery key during the reset process.
# ---------------------------------------------------------------------------
Write-Host "[3/4] Checking BitLocker status..."

try {
    $blVolumes = Get-BitLockerVolume -ErrorAction Stop
    foreach ($vol in $blVolumes) {
        if ($vol.ProtectionStatus -eq "On") {
            Write-Host "  BitLocker active on $($vol.MountPoint) - suspending for 1 reboot..."
            $isOSVolume = $vol.MountPoint -eq $env:SystemDrive
            try {
                Suspend-BitLocker -MountPoint $vol.MountPoint -RebootCount 1 -ErrorAction Stop
                Write-Host "  BitLocker suspended on $($vol.MountPoint)"
            } catch {
                if ($isOSVolume) {
                    Write-Host "  ERROR: Failed to suspend BitLocker on OS volume $($vol.MountPoint): $($_.Exception.Message)"
                    Write-Host "  Cannot proceed with wipe - BitLocker must be suspended on the OS volume to prevent recovery key prompts"
                    Stop-Transcript -ErrorAction SilentlyContinue | Out-Null
                    exit 1
                } else {
                    Write-Host "  WARNING: Failed to suspend BitLocker on $($vol.MountPoint): $($_.Exception.Message)"
                }
            }
        } else {
            Write-Host "  BitLocker already off or suspended on $($vol.MountPoint)"
        }
    }
} catch [System.Management.Automation.CommandNotFoundException] {
    Write-Host "  BitLocker module not available - skipping"
} catch {
    Write-Host "  ERROR: BitLocker state could not be determined: $($_.Exception.Message). Terminating to prevent wipe with unknown BitLocker state."
    Stop-Transcript -ErrorAction SilentlyContinue | Out-Null
    exit 1
}
Write-Host ""

# ---------------------------------------------------------------------------
# 4. Trigger device wipe via WMI bridge
# ---------------------------------------------------------------------------
# Calls the RemoteWipe CSP locally, bypassing the Fleet MDM command queue.
# This avoids server-side DB timeouts, auth errors, and command processing
# failures that caused the original issue.
#
# doWipeProtected (build 1703+): Tamper-resistant "Remove everything". If the
#   reset is interrupted (e.g. power loss), the device keeps trying until
#   complete. Removes all user data, apps, settings, and enrollment.
#
# doWipe (all builds): Standard "Remove everything". Same outcome but without
#   tamper resistance. If interrupted, the device may need manual recovery.
#
# Note: The WMI bridge requires the MDM_RemoteWipe class to be registered,
# which depends on the device having (or having had) an MDM enrollment. If
# enrollment is completely gone, the bridge call will fail.
# ---------------------------------------------------------------------------
Write-Host "[4/4] Triggering device wipe via WMI bridge..."
Write-Host "  This bypasses the MDM command channel entirely."
Write-Host ""

$namespaceName = "root\cimv2\mdm\dmmap"
$className = "MDM_RemoteWipe"
$filter = "ParentID='./Vendor/MSFT' and InstanceID='RemoteWipe'"
$wipeTriggered = $false

try {
    $session = New-CimSession -ErrorAction Stop
    $instance = Get-CimInstance -Namespace $namespaceName -ClassName $className -Filter $filter -ErrorAction Stop

    $params = New-Object Microsoft.Management.Infrastructure.CimMethodParametersCollection
    $param = [Microsoft.Management.Infrastructure.CimMethodParameter]::Create("param", "", "String", "In")
    $params.Add($param)

    $build = [int][System.Environment]::OSVersion.Version.Build

    # Try doWipeProtected first (build 1703+ / 15063+)
    if ($build -ge 15063) {
        Write-Host "  Trying doWipeProtected..."
        try {
            $result = $session.InvokeMethod($namespaceName, $instance, "doWipeProtectedMethod", $params)
            if ($result.ReturnValue -eq 0) {
                Write-Host "  Wipe command accepted (doWipeProtected)"
                $wipeTriggered = $true
            } else {
                Write-Host "  doWipeProtected returned non-zero: $($result.ReturnValue)"
            }
        } catch {
            Write-Host "  doWipeProtected failed: $($_.Exception.Message)"
        }
    }

    # Fallback to doWipe (all builds)
    if (-not $wipeTriggered) {
        Write-Host "  Trying doWipe..."
        try {
            $result = $session.InvokeMethod($namespaceName, $instance, "doWipeMethod", $params)
            if ($result.ReturnValue -eq 0) {
                Write-Host "  Wipe command accepted (doWipe)"
                $wipeTriggered = $true
            } else {
                Write-Host "  doWipe returned non-zero: $($result.ReturnValue)"
            }
        } catch {
            Write-Host "  doWipe failed: $($_.Exception.Message)"
        }
    }
} catch {
    Write-Host "  WMI bridge error: $($_.Exception.Message)"
}

Write-Host ""

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
Write-Host "=== Wipe Summary ==="
if ($wipeTriggered) {
    Write-Host "Wipe triggered. The device will reboot and reset automatically."
    Write-Host "No USB media is required."
    Write-Host "After reset the device boots to OOBE."
} else {
    Write-Host "ERROR: All wipe methods failed."
    Write-Host ""
    Write-Host "Possible causes:"
    Write-Host "  - WinRE is missing or corrupted beyond repair"
    Write-Host "  - MDM enrollment is gone (WMI bridge class not registered)"
    Write-Host "  - Component Store is damaged"
    Write-Host ""
    Write-Host "Next steps:"
    Write-Host "  - Check WinRE: reagentc /info"
    Write-Host "  - Re-enable WinRE: reagentc /enable"
    Write-Host "  - Repair Component Store: dism /Online /Cleanup-Image /RestoreHealth"
    Write-Host "  - If all else fails, a USB recovery boot is required"
    $exitCode = 1
}

Write-Host ""
Write-Host "Log saved to: $logPath"

Stop-Transcript -ErrorAction SilentlyContinue | Out-Null

exit $exitCode