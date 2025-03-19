# PowerShell script to enable the Administrator account and set a random, secure password

# Run this script as an administrator

# Function to generate a random password
function Generate-Password {
    param (
        [int]$length = 12
    )
    $chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_-+=<>?/"
    $password = -join ((1..$length) | ForEach-Object { Get-Random -Maximum $chars.length } | ForEach-Object { $chars[$_]} )
    return $password
}

# Generate a random password
$password = Generate-Password -length 12

# Convert the password to a SecureString
$securePassword = ConvertTo-SecureString $password -AsPlainText -Force

# Enable the Administrator account
Enable-LocalUser -Name "Administrator"

# Set the generated password for the Administrator account
Set-LocalUser -Name "Administrator" -Password $securePassword

# Output the password
Write-Host "Administrator account has been enabled."
Write-Host "Generated Password: $password"
