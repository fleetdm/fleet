# PowerShell script to log off all users and change their passwords

# Function to generate a random password
function Generate-Password {
    param (
        [int]$length = 12
    )
    $chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_-+=<>?/"
    $password = -join ((1..$length) | ForEach-Object { Get-Random -Maximum $chars.length } | ForEach-Object { $chars[$_]} )
    return $password
}

# Log off all non-administrative users
$loggedOffUsers = @{}
Get-WmiObject -Class Win32_UserProfile | Where-Object { $_.Special -eq $false } | ForEach-Object {
    $username = $_.LocalPath.Split('\')[-1]
    if ($username -ne "Administrator" -and $username -ne $env:USERNAME -and -not $loggedOffUsers.ContainsKey($username)) {
        try {
            $userSessions = query user | Where-Object { $_ -match "\b$username\b" }
            foreach ($session in $userSessions) {
                if ($session -match "\s+(\d+)\s+Disc\s+") {
                    # Disconnected sessions can't be logged off
                    continue
                }
                elseif ($session -match "\s+(\d+)\s+") {
                    $sessionID = $matches[1]
                    logoff $sessionID
                    $loggedOffUsers[$username] = $true
                    Write-Host "Logged out user: $username"
                }
            }
        } catch {
            Write-Host "Could not log off user: $username. Error: $($_.Exception.Message)"
        }
    }
}

# Get all local user accounts except built-in accounts like 'Administrator'
$users = Get-LocalUser | Where-Object { $_.Name -notlike "Administrator" -and $_.PrincipalSource -eq "Local" }

# Change password for each user and output the new password
foreach ($user in $users) {
    $newPassword = Generate-Password -length 12
    $securePassword = ConvertTo-SecureString $newPassword -AsPlainText -Force

    try {
        Set-LocalUser -Name $user.Name -Password $securePassword
        Write-Host "Password for user $($user.Name) changed successfully. New Password: $newPassword"
    } catch {
        Write-Host "Failed to change password for user $($user.Name)"
    }
}
