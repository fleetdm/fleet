# Please don't delete. This script is referenced in the guide here:
# https://fleetdm.com/guides/windows-mdm-setup#migrating-from-another-mdm-solution
#
# fix-windows-mdm-migration.ps1
# Comprehensive remediation script for Windows hosts migrated to Fleet from another MDM solution
# (e.g., Microsoft Intune). Each fix is gated by a detection check so it only runs if needed.
#
# This script addresses:
# 1. Incorrect MDM enrollment flag
# 2. Stale/orphaned MDM enrollment records and caches
# 3. Broken Workplace Join configuration
# 4. Unreachable WSUS server configuration
# 5. Stale EnterpriseMgmt scheduled tasks
# 6. Local account lockout caused by tattooed LocalUsersAndGroups policies
#
# Usage: Run via Fleet on affected hosts. Reboot the device after running this script.
# Reference: https://github.com/fleetdm/fleet/issues/38985

#Requires -RunAsAdministrator

# Log output for audit trail
$logPath = "$env:TEMP\fleet-mdm-migration-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
Start-Transcript -Path $logPath -ErrorAction SilentlyContinue | Out-Null

$fixesApplied = 0

Write-Host "=== Fleet Windows MDM Migration Remediation ==="
Write-Host "Log file: $logPath"
Write-Host ""

# ---------------------------------------------------------------------------
# 1. Reset MDM enrollment flag
# ---------------------------------------------------------------------------
Write-Host "[1/6] Checking MDM enrollment flag..."
$enrollmentsPath = "HKLM:\SOFTWARE\Microsoft\Enrollments"
$enrollmentFlag = (Get-ItemProperty -Path $enrollmentsPath -Name "MmpcEnrollmentFlag" -ErrorAction SilentlyContinue).MmpcEnrollmentFlag
if ($null -ne $enrollmentFlag -and 0 -ne $enrollmentFlag) {
  Write-Host "  Enrollment flag is $enrollmentFlag - resetting to 0"
  Set-ItemProperty -Path $enrollmentsPath -Name "MmpcEnrollmentFlag" -Value 0 -Type DWord
  $fixesApplied++
} else {
  Write-Host "  Enrollment flag already 0 or does not exist - skipping"
}
Write-Host ""
# ---------------------------------------------------------------------------
# 2. Remove stale MDM enrollment records, AAD cache, and MS DM Server cache
# ---------------------------------------------------------------------------
Write-Host "[2/6] Cleaning stale enrollment records and caches..."
$cachesCleaned = $false

# Clear the AAD discovery cache
$AADPath = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\CDJ\AAD"
if (Test-Path $AADPath) {
  Remove-Item -Path $AADPath -Recurse -Force
  Write-Host "  Cleared AAD discovery cache"
  $cachesCleaned = $true
  $fixesApplied++
} else {
  Write-Host "  AAD discovery cache not found - skipping"
}

# Remove GUID-based enrollment entries in failed, removed, or error states
# EnrollmentState: 0=Not enrolled, 1=Enrolled, 2=Failed, 3=Removed, 4=Error
# We preserve state 0 (could be a pending Fleet enrollment) and state 1 (active)
$EnrollmentPath = "HKLM:\SOFTWARE\Microsoft\Enrollments"
$cleaned = 0
Get-ChildItem -Path $EnrollmentPath -ErrorAction SilentlyContinue | ForEach-Object {
  if ($_.PSChildName -match '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$') {
    $state = (Get-ItemProperty -Path $_.PSPath -Name "EnrollmentState" -ErrorAction SilentlyContinue).EnrollmentState
    # Remove failed (2), removed (3), error (4) states and orphaned entries with no state
    if ($state -in @(2, 3, 4) -or $null -eq $state) {
      Write-Host "  Removing enrollment: $($_.PSChildName) (state: $state)"
      Remove-Item -Path $_.PSPath -Recurse -Force -ErrorAction SilentlyContinue
      $cleaned++
    }
  }
}
if ($cleaned -gt 0) {
  Write-Host "  Cleaned $cleaned stale enrollment entries"
  $cachesCleaned = $true
  $fixesApplied++
} else {
  Write-Host "  No stale enrollment entries found"
}

# Clear MS DM Server cache
if (Test-Path "HKLM:\SOFTWARE\Microsoft\MSDM\Server") {
  Remove-Item -Path "HKLM:\SOFTWARE\Microsoft\MSDM\Server\*" -Recurse -Force -ErrorAction SilentlyContinue
  Write-Host "  Cleared MS DM Server cache"
  $cachesCleaned = $true
  $fixesApplied++
} else {
  Write-Host "  MS DM Server cache not found - skipping"
}

# Restart Device Registration Service only if we made changes
if ($cachesCleaned) {
  Restart-Service -Name "DsSvc" -ErrorAction SilentlyContinue
  Write-Host "  Restarted Device Registration Service"
}
Write-Host ""
# ---------------------------------------------------------------------------
# 3. Fix Workplace Join configuration
# ---------------------------------------------------------------------------
Write-Host "[3/6] Checking Workplace Join configuration..."

# Re-enable Automatic-Device-Join scheduled task
$TaskPath = "\Microsoft\Windows\Workplace Join\"
$TaskName = "Automatic-Device-Join"
try {
  $task = Get-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath -ErrorAction Stop
  if ($task.State -eq "Disabled") {
    Enable-ScheduledTask -InputObject $task | Out-Null
    Write-Host "  Re-enabled Automatic-Device-Join task"
    $fixesApplied++
  } else {
    Write-Host "  Automatic-Device-Join task already enabled"
  }
} catch {
  Write-Host "  Automatic-Device-Join task not found - skipping"
}

# Configure Workplace Join policy
$WJPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows\WorkplaceJoin"
$needsUpdate = $false
if (-not (Test-Path $WJPath)) {
  $needsUpdate = $true
} else {
  $autoJoin = (Get-ItemProperty -Path $WJPath -Name "autoWorkplaceJoin" -ErrorAction SilentlyContinue).autoWorkplaceJoin
  $blockJoin = (Get-ItemProperty -Path $WJPath -Name "BlockAADWorkplaceJoin" -ErrorAction SilentlyContinue).BlockAADWorkplaceJoin
  if ($autoJoin -ne 1 -or $blockJoin -ne 0) { $needsUpdate = $true }
}
if ($needsUpdate) {
  if (-not (Test-Path $WJPath)) { New-Item -Path $WJPath -Force | Out-Null }
  Set-ItemProperty -Path $WJPath -Name "autoWorkplaceJoin" -Value 1 -Type DWord
  Set-ItemProperty -Path $WJPath -Name "BlockAADWorkplaceJoin" -Value 0 -Type DWord
  Write-Host "  Configured Workplace Join policy (autoJoin=1, BlockAAD=0)"
  $fixesApplied++
} else {
  Write-Host "  Workplace Join policy already configured correctly"
}
Write-Host ""

# ---------------------------------------------------------------------------
# 4. Remove unreachable WSUS configuration
# ---------------------------------------------------------------------------
Write-Host "[4/6] Checking WSUS configuration..."
$WUPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate"
$wuServer = $null
if (Test-Path $WUPath) {
  $wuServer = (Get-ItemProperty -Path $WUPath -Name "WUServer" -ErrorAction SilentlyContinue).WUServer
}
if ($wuServer) {
  $reachable = $false
  try {
    $null = Invoke-WebRequest -Uri $wuServer -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
    $reachable = $true
  } catch [System.Net.WebException] {
    # HTTP error responses (e.g. 403) mean the server is reachable
    if ($_.Exception.Response) { $reachable = $true }
  } catch { }

  if ($reachable) {
    Write-Host "  WSUS server $wuServer is reachable - no action taken"
  } else {
    Write-Host "  WSUS server $wuServer is unreachable - removing configuration"
    Remove-ItemProperty -Path $WUPath -Name "WUServer" -ErrorAction SilentlyContinue
    Remove-ItemProperty -Path $WUPath -Name "WUStatusServer" -ErrorAction SilentlyContinue
    Restart-Service wuauserv -Force -ErrorAction SilentlyContinue
    Write-Host "  Removed WSUS config and restarted Windows Update service"
    $fixesApplied++
  }
} else {
  Write-Host "  No WSUS server configured - skipping"
}
Write-Host ""
# ---------------------------------------------------------------------------
# 5. Unregister stale EnterpriseMgmt scheduled tasks
# ---------------------------------------------------------------------------
Write-Host "[5/6] Checking EnterpriseMgmt scheduled tasks..."
$emTasks = Get-ScheduledTask -TaskPath "\Microsoft\Windows\EnterpriseMgmt\*" -ErrorAction SilentlyContinue
if ($emTasks) {
  $emTasks | Unregister-ScheduledTask -Confirm:$false -ErrorAction SilentlyContinue
  Write-Host "  Unregistered $($emTasks.Count) stale EnterpriseMgmt tasks"
  $fixesApplied++
} else {
  Write-Host "  No EnterpriseMgmt tasks found - skipping"
}
Write-Host ""

# ---------------------------------------------------------------------------
# 6. Fix local account lockout from tattooed LocalUsersAndGroups policies
# ---------------------------------------------------------------------------
Write-Host "[6/6] Checking for tattooed LocalUsersAndGroups policies..."

$lockoutFixed = $false

# Check for orphaned LocalUsersAndGroups in PolicyManager current device path
$lugCurrentPath = "HKLM:\SOFTWARE\Microsoft\PolicyManager\current\device\LocalUsersAndGroups"
if (Test-Path $lugCurrentPath) {
  Remove-Item -Path $lugCurrentPath -Recurse -Force -ErrorAction SilentlyContinue
  Write-Host "  Removed orphaned LocalUsersAndGroups from PolicyManager current device"
  $lockoutFixed = $true
}

# Check for orphaned LocalUsersAndGroups in PolicyManager provider paths
$providersPath = "HKLM:\SOFTWARE\Microsoft\PolicyManager\Providers"
if (Test-Path $providersPath) {
  Get-ChildItem -Path $providersPath -ErrorAction SilentlyContinue | ForEach-Object {
    $lugProviderPath = Join-Path $_.PSPath "default\Device\LocalUsersAndGroups"
    if (Test-Path $lugProviderPath) {
      Remove-Item -Path $lugProviderPath -Recurse -Force -ErrorAction SilentlyContinue
      Write-Host "  Removed orphaned LocalUsersAndGroups from provider: $($_.PSChildName)"
      $lockoutFixed = $true
    }
  }
}

if ($lockoutFixed) {
  # Reset SeInteractiveLogonRight to Windows defaults
  # Default: Administrators (S-1-5-32-544), Users (S-1-5-32-545), Backup Operators (S-1-5-32-551)
  $tempCfg = "$env:TEMP\secpol_fix.cfg"
  $tempDb = "$env:TEMP\secpol_fix.sdb"
  try {
    # Export current security policy
    secedit /export /cfg $tempCfg /quiet 2>$null

    # Read, patch, and write the config
    $content = Get-Content $tempCfg -Raw
    $defaultRight = "*S-1-5-32-544,*S-1-5-32-545,*S-1-5-32-551"
    if ($content -match "SeInteractiveLogonRight") {
      $content = $content -replace "SeInteractiveLogonRight\s*=.*", "SeInteractiveLogonRight = $defaultRight"
    } else {
      $content = $content -replace "(\[Privilege Rights\])", "$1`r`nSeInteractiveLogonRight = $defaultRight"
    }
    Set-Content -Path $tempCfg -Value $content

    # Apply the patched config
    secedit /configure /db $tempDb /cfg $tempCfg /quiet 2>$null
    Write-Host "  Reset SeInteractiveLogonRight to defaults (Administrators, Users, Backup Operators)"

    $fixesApplied++
  } catch {
    Write-Host "  Warning: Failed to reset SeInteractiveLogonRight - $_"
  } finally {
    Remove-Item $tempCfg -Force -ErrorAction SilentlyContinue
    Remove-Item $tempDb -Force -ErrorAction SilentlyContinue
  }
} else {
  Write-Host "  No orphaned LocalUsersAndGroups policies found - skipping"
}
Write-Host ""

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
Write-Host "=== Remediation Complete ==="
Write-Host "Fixes applied: $fixesApplied"
if ($fixesApplied -gt 0) {
  Write-Host ""
  Write-Host "IMPORTANT: Reboot the device now to apply changes."
  Write-Host "After reboot, select Refetch on the host details page in Fleet."
} else {
  Write-Host "No issues detected - device appears healthy."
}
Write-Host "Log saved to: $logPath"

Stop-Transcript -ErrorAction SilentlyContinue | Out-Null
