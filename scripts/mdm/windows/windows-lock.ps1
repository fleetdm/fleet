# PowerShell script to log off all non-administrative users and disable their accounts

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

# Disable all non-administrative local user accounts
Get-LocalUser | Where-Object { $_.Enabled -eq $true -and $_.Name -ne "Administrator" } | ForEach-Object {
    $username = $_.Name
    Disable-LocalUser -Name $username
    Write-Host "Disabled account for $username"
}

Write-Host "All non-administrative users have been logged out and their accounts disabled."
