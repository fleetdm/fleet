# PowerShell script to log off all users and block all logons

# Log off all users
$loggedOffUsers = @{}
Get-WmiObject -Class Win32_UserProfile | Where-Object { $_.Special -eq $false } | ForEach-Object {
    $username = $_.LocalPath.Split('\')[-1]
    if ($username -ne $env:USERNAME -and -not $loggedOffUsers.ContainsKey($username)) {
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

# Deny local and Remote Desktop logon for the Users (S-1-5-32-545) and Guests
# (S-1-5-32-546) groups. This blocks every account type (local, AD, Azure/Entra)
# without touching cached logins, and is undone by windows_unlock.ps1.
function Set-DenyLogonRights {
    param([string[]]$Sids)

    $id = [guid]::NewGuid().ToString()
    $cfg = Join-Path $env:TEMP "fleet_user_rights_$id.inf"
    $db = Join-Path $env:TEMP "fleet_user_rights_$id.sdb"
    $value = ($Sids | ForEach-Object { "*$_" }) -join ','

    try {
        secedit /export /cfg $cfg /areas USER_RIGHTS | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "secedit export failed with exit code $LASTEXITCODE" }

        $lines = @(Get-Content -Path $cfg)
        foreach ($right in 'SeDenyInteractiveLogonRight', 'SeDenyRemoteInteractiveLogonRight') {
            $newLine = "$right = $value"
            if ($lines -match "^$right\s*=") {
                $lines = $lines | ForEach-Object { if ($_ -match "^$right\s*=") { $newLine } else { $_ } }
            } else {
                $lines = $lines | ForEach-Object { $_; if ($_ -eq '[Privilege Rights]') { $newLine } }
            }
        }
        $lines | Set-Content -Path $cfg -Encoding Unicode

        secedit /configure /db $db /cfg $cfg /areas USER_RIGHTS | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "secedit configure failed with exit code $LASTEXITCODE" }
    } finally {
        Remove-Item -Path $cfg, $db -Force -ErrorAction SilentlyContinue
    }
}

try {
    Set-DenyLogonRights -Sids 'S-1-5-32-545', 'S-1-5-32-546'
    Write-Host "Logon denied for the Users and Guests groups."
} catch {
    Write-Host "Could not deny logon: $($_.Exception.Message)"
    exit 1
}

# The deny-logon policy above likely makes the next two steps redundant. They
# are kept as extra protection.

# Disable all local user accounts
Get-LocalUser | Where-Object { $_.Enabled -eq $true } | ForEach-Object {
    $username = $_.Name
    Disable-LocalUser -Name $username
    Write-Host "Disabled account for $username"
}

# Disable additional AD logins
New-ItemProperty -Path "HKLM:\Software\Microsoft\PolicyManager\default\Settings\AllowSignInOptions" -Name 'value' -Value 3 -PropertyType DWORD -Force

Write-Host "All users have been logged out and their accounts disabled."
Write-Host "Logging in with other Microsoft accounts has been disabled"

# Shutdown computer in 15 seconds, after command has returned to Fleet
Write-Host "Shutting down in 15 seconds"
shutdown /s /f /t 15
