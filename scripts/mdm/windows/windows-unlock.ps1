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

# Re-enable additional AD logins
New-ItemProperty -Path "HKLM:\Software\Microsoft\PolicyManager\default\Settings\AllowSignInOptions" -Name 'value' -Value 0 -PropertyType DWORD -Force

# Re-enable cached logins for AD/Azure/Entra accounts
New-ItemProperty -Path "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Winlogon\" -Name 'CachedLogonsCount' -Value 10 -PropertyType String -Force
