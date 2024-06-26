# PowerShell script to disable the Administrator account

# Run this script as an administrator

# Disable the Administrator account
Disable-LocalUser -Name "Administrator"

Write-Host "Administrator account has been disabled."
