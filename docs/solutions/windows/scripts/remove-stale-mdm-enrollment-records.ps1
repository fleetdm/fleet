# Please don't delete. This script is referenced in the guide here: https://fleetdm.com/guides/windows-mdm-setup#migrating-from-another-mdm-solution
# Removes stale MDM enrollment registry entries, AAD discovery cache, and MS DM Server cache
# that can block Fleet MDM enrollment after migrating from another MDM solution.
# Reboot the device after running this script.

# 1. Clear the AAD discovery cache
$AADPath = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\CDJ\AAD"
if (Test-Path $AADPath) {
  Remove-Item -Path $AADPath -Recurse -Force
  Write-Host "Cleared AAD discovery cache"
} else {
  Write-Host "AAD discovery cache not found - skipping"
}

# 2. Remove stale GUID-based enrollment entries (failed, removed, or error states)
$EnrollmentPath = "HKLM:\SOFTWARE\Microsoft\Enrollments"
$cleaned = 0
Get-ChildItem -Path $EnrollmentPath -ErrorAction SilentlyContinue | ForEach-Object {
  if ($_.PSChildName -match '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$') {
    $state = (Get-ItemProperty -Path $_.PSPath -Name "EnrollmentState" -ErrorAction SilentlyContinue).EnrollmentState
    # EnrollmentState: 0=Not enrolled, 1=Enrolled, 2=Failed, 3=Removed, 4=Failed (old may still work)
    if ($state -in @(2, 3, 4)) {
      Remove-Item -Path $_.PSPath -Recurse -Force -ErrorAction SilentlyContinue
      Write-Host "Removed stale enrollment: $($_.PSChildName) (state: $state)"
      $cleaned++
    }
  }
}
Write-Host "Cleaned $cleaned stale enrollment entries"

# 3. Clear MS DM Server cache
if (Test-Path "HKLM:\SOFTWARE\Microsoft\MSDM\Server") {
  Remove-Item -Path "HKLM:\SOFTWARE\Microsoft\MSDM\Server\*" -Recurse -Force -ErrorAction SilentlyContinue
  Write-Host "Cleared MS DM Server cache"
} else {
  Write-Host "MS DM Server cache not found - skipping"
}

# 4. Restart Device Registration Service
Restart-Service -Name "DsSvc" -ErrorAction SilentlyContinue
Write-Host "Restarted Device Registration Service"
