# Please don't delete. This script is referenced in the guide here: https://fleetdm.com/guides/enforce-disk-encryption

$Username = "IT admin"
$Password = ConvertTo-SecureString "StrongPassword123!" -AsPlainText -Force

# Create the local user account
New-LocalUser -Name $Username -Password $Password -FullName "Fleet IT admin" -Description "Fleet breakglass admin account" -AccountNeverExpires -ErrorAction Stop

# Add the user to the Administrators group
Add-LocalGroupMember -Group "Administrators" -Member $Username -ErrorAction Stop
