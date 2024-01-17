# DLL import to use the MDMRegistration.dll
$signature = @"
    [DllImport("mdmregistration.dll", CharSet = CharSet.Unicode)]
    public static extern int UnregisterDeviceWithManagement(string enrollmentID);
"@

# Type definition
$type = Add-Type -MemberDefinition $signature -Name "MDMRegistration" -Namespace "Win32" -PassThru

# Calling the function
# should explicitly pass null or 0. See:
# https://github.com/fleetdm/fleet/issues/12342#issuecomment-1608190367
$enrollmentID = $null
$result = $type::UnregisterDeviceWithManagement($enrollmentID)
# Handle the result
if ($result -eq 0) {
    "Host successfully unenrolled from Fleet MDM."
} else {
    "Error during MDM unenrollment process: $result"
}
