# PowerShell script to enable all disabled local user accounts

# Get all local user accounts
$localUsers = Get-LocalUser

# Enable each disabled user account
foreach ($user in $localUsers) {
    if ($user.Enabled -eq $false) {
        Enable-LocalUser -Name $user.Name
        Write-Host "Enabled user account: $($user.Name)"
    }
}

Write-Host "All disabled user accounts have been enabled."
