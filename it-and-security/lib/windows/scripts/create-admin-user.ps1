# Please don't delete. This script is referenced in the guide here: https://fleetdm.com/guides/enforce-disk-encryption
#
# Requires a FLEET_SECRET_BREAKGLASS_ADMIN_PASSWORD custom variable in Fleet (Controls > Variables).
# Fleet substitutes it when the script is sent to the host, so the password never lives in this repo.

$Username = "IT admin"
# Single-quoted so PowerShell doesn't expand $ or backticks in the substituted value; the password must not contain a single quote.
$Password = ConvertTo-SecureString '$FLEET_SECRET_BREAKGLASS_ADMIN_PASSWORD' -AsPlainText -Force

# Create the local user account
New-LocalUser -Name $Username -Password $Password -FullName "Fleet IT admin" -Description "Fleet breakglass admin account" -AccountNeverExpires -ErrorAction Stop

# Add the user to the Administrators group
Add-LocalGroupMember -Group "Administrators" -Member $Username -ErrorAction Stop
