<#
.SYNOPSIS
    Fleet Deployment Script: Uninstall & Clean Up Toast Notification System
    Deploy via Fleet: Controls > Scripts > Add script

.DESCRIPTION
    Full cleanup of the toast notification PoC. Removes:
    - The FleetWindowsUpdateToast scheduled task
    - The deployed script directory ($env:ProgramData\Fleet\ToastNotification)
    - Active toast notifications from the Action Center (best-effort)

    Safe to run even if the task was never installed. Idempotent.

.NOTES
    Run as Administrator (or via Fleet, which runs as SYSTEM).
#>

$TaskName  = "FleetWindowsUpdateToast"
$ScriptDir = "$env:ProgramData\Fleet\ToastNotification"

Write-Host "=== Fleet Toast Notification Cleanup ==="
Write-Host ""

# ============================================================
# 1. STOP AND REMOVE THE SCHEDULED TASK
# ============================================================

$existingTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($existingTask) {
    # Stop the task if it's currently running
    if ($existingTask.State -eq 'Running') {
        Stop-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
        Write-Host "Stopped running task: $TaskName"
    }
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
    Write-Host "Removed scheduled task: $TaskName"
}
else {
    Write-Host "Scheduled task '$TaskName' not found. Skipping."
}

# ============================================================
# 2. REMOVE DEPLOYED SCRIPT FILES
# ============================================================

if (Test-Path $ScriptDir) {
    Remove-Item -Path $ScriptDir -Recurse -Force
    Write-Host "Removed script directory: $ScriptDir"
}
else {
    Write-Host "Script directory not found. Skipping."
}

# Clean up parent Fleet directory if empty
$fleetDir = "$env:ProgramData\Fleet"
if ((Test-Path $fleetDir) -and @(Get-ChildItem $fleetDir -ErrorAction SilentlyContinue).Count -eq 0) {
    Remove-Item -Path $fleetDir -Force
    Write-Host "Removed empty Fleet directory: $fleetDir"
}

# ============================================================
# 3. CLEAR TOAST NOTIFICATIONS FROM ACTION CENTER
# ============================================================

try {
    [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
    $AppId = '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'
    $history = [Windows.UI.Notifications.ToastNotificationManager]::History
    $history.Clear($AppId)
    Write-Host "Cleared toast notification history for PowerShell AppId."
}
catch {
    Write-Host "Could not clear toast history (may need user context): $_"
    Write-Host "  Toast notifications in Action Center may need to be dismissed manually."
}

# ============================================================
# DONE
# ============================================================

Write-Host ""
Write-Host "Cleanup complete. VM is reset and ready for a fresh install."
exit 0
