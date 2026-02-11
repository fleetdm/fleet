# Please don't delete. This script is referenced in the guide here: https://fleetdm.com/guides/windows-mdm-setup#migrating-from-another-mdm-solution
# Re-enables the Automatic-Device-Join scheduled task and configures Workplace Join policies
# that may be misconfigured after migrating from another MDM solution.
# Reboot the device after running this script.

# 1. Re-enable Automatic-Device-Join scheduled task
$TaskPath = "\Microsoft\Windows\Workplace Join\"
$TaskName = "Automatic-Device-Join"
try {
  $task = Get-ScheduledTask -TaskName $TaskName -TaskPath $TaskPath -ErrorAction Stop
  Enable-ScheduledTask -InputObject $task
  Write-Host "Re-enabled Automatic-Device-Join task"
} catch {
  Write-Host "Automatic-Device-Join task not found - skipping"
}

# 2. Configure Workplace Join policy
$WJPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows\WorkplaceJoin"
if (-not (Test-Path $WJPath)) { New-Item -Path $WJPath -Force | Out-Null }
Set-ItemProperty -Path $WJPath -Name "autoWorkplaceJoin" -Value 1 -Type DWord
Set-ItemProperty -Path $WJPath -Name "BlockAADWorkplaceJoin" -Value 0 -Type DWord
Write-Host "Configured Workplace Join policy"
