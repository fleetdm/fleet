# Please don't delete. This script is referenced in the guide here: https://fleetdm.com/guides/windows-mdm-setup#migrating-from-another-mdm-solution
# Resets the MmpcEnrollmentFlag registry value that can prevent Fleet from reporting
# MDM status correctly after migrating from another MDM solution (e.g., Intune).
# Reboot the device after running this script.

$enrollmentsPath = "HKLM:\SOFTWARE\Microsoft\Enrollments"
$enrollmentFlag = (Get-ItemProperty -Path $enrollmentsPath -Name "MmpcEnrollmentFlag" -ErrorAction SilentlyContinue).MmpcEnrollmentFlag
if ($null -ne $enrollmentFlag -and 0 -ne $enrollmentFlag) {
  Write-Host "Enrollment flag current value $enrollmentFlag - setting to 0"
  Set-ItemProperty -Path $enrollmentsPath -Name "MmpcEnrollmentFlag" -Value 0 -Type DWord
} else {
  Write-Host "Enrollment flag already 0 or does not exist"
}
