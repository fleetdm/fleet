# PowerShell script to undo windows_lock.ps1 and allow logons again

# Restore the default deny-logon policy (Guests group only) for local and
# Remote Desktop logon.
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
    Set-DenyLogonRights -Sids 'S-1-5-32-546'
    Write-Host "Logon allowed again for the Users group."
} catch {
    Write-Host "Could not restore logon rights: $($_.Exception.Message)"
    exit 1
}

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

# Lock no longer changes this, but hosts locked by earlier versions of
# windows_lock.ps1 have cached logins turned off, so restore them here.
New-ItemProperty -Path "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Winlogon\" -Name 'CachedLogonsCount' -Value 10 -PropertyType String -Force
