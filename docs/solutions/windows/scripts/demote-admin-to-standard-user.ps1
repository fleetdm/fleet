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

# S-1-5-32-544 is the well-known SID for the built-in Administrators group. Its display
# name is localized on non-English Windows installations, so the SID is used instead.
$administratorsSid = "S-1-5-32-544"

# explorer.exe runs once per interactive desktop session (console or RDP), so this finds
# the signed-in user regardless of session type. Win32_ComputerSystem.UserName only
# reports the physical console session and would miss or misidentify an RDP session.
$explorerProcesses = Get-Process -Name explorer -IncludeUserName -ErrorAction SilentlyContinue

if (-not $explorerProcesses) {
    Write-Output "No interactive user session found on this host. Sign in as the end user, then re-run this script."
    exit 1
}

$signedInUsers = @($explorerProcesses | Select-Object -ExpandProperty UserName -Unique)

if ($signedInUsers.Count -gt 1) {
    Write-Output "More than one interactive user session was found ($($signedInUsers -join ', ')). Sign out all but the end user's session, then re-run this script."
    exit 1
}

$currentUser = $signedInUsers[0]

try {
    $admin = Get-LocalGroupMember -SID $administratorsSid -ErrorAction Stop | Where-Object { $_.Name -eq $currentUser }
} catch {
    Write-Output "Failed to read the local Administrators group: $_"
    exit 1
}

if (-not $admin) {
    Write-Output "$currentUser is not a member of the local Administrators group. Nothing to do."
    exit 0
}

try {
    Remove-LocalGroupMember -SID $administratorsSid -Member $currentUser -ErrorAction Stop
} catch {
    Write-Output "Failed to remove $currentUser from the local Administrators group: $_"
    exit 1
}

Write-Output "Removed $currentUser from the local Administrators group. Sign out and back in for the change to take effect."
