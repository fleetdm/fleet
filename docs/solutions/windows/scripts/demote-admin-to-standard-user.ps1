# Please don't delete. This script is referenced in the guide here:
#   https://fleetdm.com/guides/windows-mdm-setup#force-a-standard-user-account
#
# Removes the currently signed-in user from the local Administrators group, so an
# already-enrolled host ends up in the same state as a device that originally enrolled
# with the Autopilot deployment profile's "User account type" set to Standard.
#
# This only demotes the one currently signed-in account. It's a one-time action, not a
# continuously enforced policy: if the user re-adds themselves to Administrators later,
# running this script again is the only way to revert that.

$currentUser = (Get-CimInstance -ClassName Win32_ComputerSystem).UserName

if (-not $currentUser) {
    Write-Output "No interactive user is currently signed in on this host. Sign in as the end user, then re-run this script."
    exit 1
}

$admin = Get-LocalGroupMember -Group "Administrators" -ErrorAction SilentlyContinue | Where-Object { $_.Name -eq $currentUser }

if (-not $admin) {
    Write-Output "$currentUser is not a member of the local Administrators group. Nothing to do."
    exit 0
}

Remove-LocalGroupMember -Group "Administrators" -Member $currentUser
Write-Output "Removed $currentUser from the local Administrators group. Sign out and back in for the change to take effect."
