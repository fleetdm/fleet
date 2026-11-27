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

# Enable the given accounts. Returns $true only if every one was enabled.
function Enable-Accounts {
    param($Users)

    $ok = $true
    foreach ($user in $Users) {
        try {
            Enable-LocalUser -SID $user.SID -ErrorAction Stop
            Write-Host "Enabled user account: $($user.Name)"
        } catch {
            Write-Host "Could not enable user account $($user.Name): $($_.Exception.Message)"
            $ok = $false
        }
    }
    $ok
}

$signInKey = "HKLM:\Software\Microsoft\PolicyManager\default\Settings\AllowSignInOptions"

# Put the sign-in options value back. 1 is the default ("allowed"); 0 means
# "not allowed" and grays out Settings > Accounts > Sign-in options. Pass $null
# to use the default, or an empty string to remove the value.
function Restore-SignInOptions {
    param($Original)

    if ($Original -eq '') {
        Remove-ItemProperty -Path $signInKey -Name 'value' -ErrorAction Stop
    } else {
        $value = if ($null -eq $Original) { 1 } else { [int]$Original }
        New-ItemProperty -Path $signInKey -Name 'value' -Value $value -PropertyType DWORD -Force -ErrorAction Stop | Out-Null
    }
    Write-Host "Restored sign-in options."
}

# Restore what windows_lock.ps1 saved: the two deny-logon policies and the
# local accounts it disabled. The saved record is only deleted once everything
# is restored, so a failed unlock can be retried.
$stateKey = "HKLM:\SOFTWARE\Fleet\WindowsLock"
$restored = $true
$localUsers = @()
try {
    $localUsers = @(Get-LocalUser -ErrorAction Stop)
} catch {
    Write-Host "Could not list local accounts: $($_.Exception.Message)"
    $restored = $false
}

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
    if (-not (Enable-Accounts -Users @($localUsers | Where-Object { $savedSids -contains $_.SID.Value -and -not $_.Enabled }))) {
        $restored = $false
    }

    # Locks from before this value was saved have none, so use the default.
    try {
        Restore-SignInOptions -Original $saved.SignInOptions
    } catch {
        Write-Host "Could not restore sign-in options: $($_.Exception.Message)"
        $restored = $false
    }

    if ($restored) { Remove-Item -Path $stateKey -Recurse -Force }
} else {
    # Locked by an older version of windows_lock.ps1, which didn't change the
    # deny-logon policies or keep a record of the accounts it disabled. It did
    # turn off cached logins (set to 0), which is how to tell. Enable every
    # disabled account except the ones Windows never enables on its own: Guest
    # (-501), DefaultAccount (-503), WDAGUtilityAccount (-504).
    $cachedLogons = (Get-ItemProperty -Path "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Winlogon" -Name 'CachedLogonsCount' -ErrorAction SilentlyContinue).CachedLogonsCount
    if ($cachedLogons -eq '0') {
        $enabled = Enable-Accounts -Users @($localUsers | Where-Object { -not $_.Enabled -and $_.SID.Value -notmatch '-(501|503|504)$' })
        # Changing this ends the old-lock fallback, so only do it once every account is back.
        if ($enabled -and $restored) {
            New-ItemProperty -Path "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Winlogon\" -Name 'CachedLogonsCount' -Value 10 -PropertyType String -Force
        } else {
            $restored = $false
        }
    }

    try {
        Restore-SignInOptions -Original $null
    } catch {
        Write-Host "Could not restore sign-in options: $($_.Exception.Message)"
        $restored = $false
    }
}

if (-not $restored) {
    Write-Host "Unlock was incomplete. Run it again to retry."
    exit 1
}
