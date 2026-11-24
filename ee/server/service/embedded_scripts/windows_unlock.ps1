# PowerShell script to undo windows_lock.ps1 and allow logons again

$denyRights = 'SeDenyInteractiveLogonRight', 'SeDenyRemoteInteractiveLogonRight'

# Set the "Deny log on locally" and "Deny log on through Remote Desktop
# Services" policies.
function Set-DenyLogonRights {
    param([hashtable]$Values)

    $id = [guid]::NewGuid().ToString()
    $cfg = Join-Path $env:TEMP "fleet_user_rights_$id.inf"
    $db = Join-Path $env:TEMP "fleet_user_rights_$id.sdb"

    try {
        secedit /export /cfg $cfg /areas USER_RIGHTS | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "secedit export failed with exit code $LASTEXITCODE" }

        $lines = @(Get-Content -Path $cfg)
        foreach ($right in $denyRights) {
            $newLine = "$right = $($Values[$right])"
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

# Restore what windows_lock.ps1 saved: the two deny-logon policies and the
# local accounts it disabled.
$stateKey = "HKLM:\SOFTWARE\Fleet\WindowsLock"
$localUsers = @(Get-LocalUser)

if (Test-Path $stateKey) {
    $saved = Get-ItemProperty -Path $stateKey

    if ($null -ne $saved.DenyInteractive -and $null -ne $saved.DenyRemote) {
        try {
            Set-DenyLogonRights -Values @{ SeDenyInteractiveLogonRight = $saved.DenyInteractive; SeDenyRemoteInteractiveLogonRight = $saved.DenyRemote }
            Write-Host "Restored the deny-logon policies."
        } catch {
            Write-Host "Could not restore logon rights: $($_.Exception.Message)"
            exit 1
        }
    }

    $savedSids = @($saved.DisabledAccounts)
    foreach ($user in @($localUsers | Where-Object { $savedSids -contains $_.SID.Value -and -not $_.Enabled })) {
        Enable-LocalUser -SID $user.SID
        Write-Host "Enabled user account: $($user.Name)"
    }

    Remove-Item -Path $stateKey -Recurse -Force
} else {
    # Locked by an older version of windows_lock.ps1, which didn't change the
    # deny-logon policies or keep a record of the accounts it disabled. It did
    # turn off cached logins (set to 0), which is how to tell. Enable every
    # disabled account except the ones Windows never enables on its own: Guest
    # (-501), DefaultAccount (-503), WDAGUtilityAccount (-504).
    $cachedLogons = (Get-ItemProperty -Path "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Winlogon" -Name 'CachedLogonsCount' -ErrorAction SilentlyContinue).CachedLogonsCount
    if ($cachedLogons -eq '0') {
        New-ItemProperty -Path "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Winlogon\" -Name 'CachedLogonsCount' -Value 10 -PropertyType String -Force

        foreach ($user in @($localUsers | Where-Object { -not $_.Enabled -and $_.SID.Value -notmatch '-(501|503|504)$' })) {
            Enable-LocalUser -SID $user.SID
            Write-Host "Enabled user account: $($user.Name)"
        }
    }
}

# Re-enable additional AD logins. 1 is the default ("allowed"); 0 means "not
# allowed" and grays out Settings > Accounts > Sign-in options.
New-ItemProperty -Path "HKLM:\Software\Microsoft\PolicyManager\default\Settings\AllowSignInOptions" -Name 'value' -Value 1 -PropertyType DWORD -Force
