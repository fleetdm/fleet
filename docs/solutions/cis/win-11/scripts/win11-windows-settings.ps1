#Requires -RunAsAdministrator
<#
.SYNOPSIS
    CIS Windows 11 Enterprise Benchmark v4.0.0 - Security Registry Settings Remediation

.DESCRIPTION
    Configures common security registry settings to meet CIS Windows 11 Enterprise Benchmark v4.0.0
    requirements. This script is idempotent - safe to run multiple times.

.NOTES
    CIS Benchmark Controls Covered (registry-based, Section 2-19):
    - 2.3.1.4  Block Microsoft accounts                            -> 3
    - 2.3.1.5  Limit blank password use to console logon only      -> 1 (Enabled)
    - 2.3.2.1  Audit: Force audit policy subcategory settings      -> 1 (Enabled)
    - 2.3.4.1  Devices: Prevent users from installing printer drivers -> 1 (Enabled)
    - 2.3.7.1  Interactive logon: Do not require CTRL+ALT+DEL     -> 0 (Disabled)
    - 2.3.7.2  Interactive logon: Don't display last username      -> 1 (Enabled)
    - 2.3.10.1 Network access: Allow anonymous SID/name translation -> 0 (Disabled)
    - 2.3.10.2 Network access: Do not allow anonymous enum of SAM  -> 1 (Enabled)
    - 2.3.10.3 Network access: Do not allow anonymous enum of SAM and shares -> 1 (Enabled)
    - 2.3.11.4 Network security: Do not store LAN Manager hash     -> 1 (Enabled)
    - 2.3.17.1 UAC: Admin approval mode for Built-in Administrator -> 1 (Enabled)
    - 2.3.17.2 UAC: Behavior of elevation prompt for admins        -> 2 (Prompt for credentials)
    - 2.3.17.5 UAC: Behavior of elevation prompt for standard users -> 0 (Auto-deny)
    - 2.3.17.6 UAC: Run all administrators in Admin Approval Mode  -> 1 (Enabled)
    - 18.1.1.1 Prevent enabling lock screen camera                 -> 1 (Enabled)
    - 18.1.1.2 Prevent enabling lock screen slide show             -> 1 (Enabled)
    - 18.3.3   Configure SMB v1 client driver                      -> Disabled
    - 18.3.4   Configure SMB v1 server                             -> Disabled
    - 18.4.1   MSS: AutoAdminLogon                                 -> "0" (Disabled)
    - 18.9.3   Autoplay: DisableAutoplay                           -> 1 (Enabled)
    - 18.9.4   AutoRun: NoDriveTypeAutoRun                         -> 255 (All drives)
    - 18.9.48  Windows Installer: AlwaysInstallElevated            -> 0 (Disabled)
    - 18.9.75  WinRM: AllowUnencryptedTraffic                      -> 0 (Disabled)

    Requires: Administrator privileges
    Note: Changes take effect immediately for registry settings. Some may require
    a reboot or Group Policy update to fully apply.
#>

[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Write-Status {
    param([string]$Message, [string]$Level = 'INFO')
    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
    Write-Output "[$timestamp] [$Level] $Message"
}

function Set-RegistryValue {
    param(
        [string]$Path,
        [string]$Name,
        [object]$Value,
        [string]$Type = 'DWord',
        [string]$Description = ''
    )

    try {
        # Create registry key if it does not exist
        if (-not (Test-Path $Path)) {
            $null = New-Item -Path $Path -Force
            Write-Status "  Created key: $Path"
        }

        $current = Get-ItemProperty -Path $Path -Name $Name -ErrorAction SilentlyContinue
        if ($null -ne $current -and $current.$Name -eq $Value) {
            Write-Status "  [OK] $Path\$Name = $Value (already set)"
            return
        }

        Set-ItemProperty -Path $Path -Name $Name -Value $Value -Type $Type -Force
        Write-Status "  [SET] $Path\$Name = $Value  $Description"
    } catch {
        Write-Status "  [FAIL] $Path\$Name : $_" 'WARN'
    }
}

Write-Status "Starting CIS Windows security registry settings remediation"

# --- Section 2.3: Security Options ---

# 2.3.1.4: Block Microsoft accounts = 3 (Users can't add or log on with Microsoft accounts)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name  'NoConnectedUser' `
    -Value 3 `
    -Description '(CIS 2.3.1.4)'

# 2.3.1.5: Limit blank password use to console logon only = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' `
    -Name  'LimitBlankPasswordUse' `
    -Value 1 `
    -Description '(CIS 2.3.1.5)'

# 2.3.2.1: Audit: Force audit policy subcategory settings = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' `
    -Name  'SCENoApplyLegacyAuditPolicy' `
    -Value 1 `
    -Description '(CIS 2.3.2.1)'

# 2.3.4.1: Devices: Prevent users from installing printer drivers = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SYSTEM\CurrentControlSet\Control\Print\Providers\LanMan Print Services\Servers' `
    -Name  'AddPrinterDrivers' `
    -Value 1 `
    -Description '(CIS 2.3.4.1)'

# 2.3.7.1: Interactive logon: Do not require CTRL+ALT+DEL = 0 (Disabled = CTRL+ALT+DEL required)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name  'DisableCAD' `
    -Value 0 `
    -Description '(CIS 2.3.7.1)'

# 2.3.7.2: Interactive logon: Don't display last username = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name  'DontDisplayLastUserName' `
    -Value 1 `
    -Description '(CIS 2.3.7.2)'

# 2.3.10.1: Network access: Allow anonymous SID/Name translation = 0 (Disabled)
Set-RegistryValue `
    -Path  'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' `
    -Name  'TurnOffAnonymousBlock' `
    -Value 0 `
    -Description '(CIS 2.3.10.1)'

# 2.3.10.2: Network access: Do not allow anonymous enumeration of SAM accounts = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' `
    -Name  'RestrictAnonymousSAM' `
    -Value 1 `
    -Description '(CIS 2.3.10.2)'

# 2.3.10.3: Network access: Do not allow anonymous enumeration of SAM accounts and shares = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' `
    -Name  'RestrictAnonymous' `
    -Value 1 `
    -Description '(CIS 2.3.10.3)'

# 2.3.11.4: Network security: Do not store LAN Manager hash on next password change = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' `
    -Name  'NoLMHash' `
    -Value 1 `
    -Description '(CIS 2.3.11.4)'

# --- Section 2.3.17: UAC Settings ---

# 2.3.17.1: UAC: Admin Approval Mode for Built-in Administrator = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name  'FilterAdministratorToken' `
    -Value 1 `
    -Description '(CIS 2.3.17.1)'

# 2.3.17.2: UAC: Behavior of elevation prompt for administrators = 2 (Prompt for credentials on the secure desktop)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name  'ConsentPromptBehaviorAdmin' `
    -Value 2 `
    -Description '(CIS 2.3.17.2)'

# 2.3.17.5: UAC: Behavior of elevation prompt for standard users = 0 (Automatically deny elevation requests)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name  'ConsentPromptBehaviorUser' `
    -Value 0 `
    -Description '(CIS 2.3.17.5)'

# 2.3.17.6: UAC: Run all administrators in Admin Approval Mode = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name  'EnableLUA' `
    -Value 1 `
    -Description '(CIS 2.3.17.6)'

# --- Section 18: Administrative Templates ---

# 18.1.1.1: Prevent enabling lock screen camera = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Personalization' `
    -Name  'NoLockScreenCamera' `
    -Value 1 `
    -Description '(CIS 18.1.1.1)'

# 18.1.1.2: Prevent enabling lock screen slide show = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Personalization' `
    -Name  'NoLockScreenSlideshow' `
    -Value 1 `
    -Description '(CIS 18.1.1.2)'

# 18.3.3: SMBv1 Client Driver = Disabled (4 = Disabled)
Set-RegistryValue `
    -Path  'HKLM:\SYSTEM\CurrentControlSet\Services\MrxSmb10' `
    -Name  'Start' `
    -Value 4 `
    -Description '(CIS 18.3.3 - SMBv1 client disabled)'

# 18.3.4: SMBv1 Server = Disabled (0)
Set-RegistryValue `
    -Path  'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters' `
    -Name  'SMB1' `
    -Value 0 `
    -Description '(CIS 18.3.4 - SMBv1 server disabled)'

# 18.4.1: MSS: Disable AutoAdminLogon (REG_SZ value)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon' `
    -Name  'AutoAdminLogon' `
    -Value '0' `
    -Type  'String' `
    -Description '(CIS 18.4.1 - Disable AutoAdminLogon)'

# 18.9.3: Disable Autoplay for all drives = 1 (Enabled)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\Explorer' `
    -Name  'NoDriveTypeAutoRun' `
    -Value 255 `
    -Description '(CIS 18.9.3/4 - Disable AutoRun all drives)'

# 18.9.4: AutoRun commands are disabled
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\Explorer' `
    -Name  'NoAutorun' `
    -Value 1 `
    -Description '(CIS 18.9.4 - Disable AutoRun commands)'

# 18.9.48: Windows Installer: Always install with elevated privileges = 0 (Disabled)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Installer' `
    -Name  'AlwaysInstallElevated' `
    -Value 0 `
    -Description '(CIS 18.9.48 - Disable AlwaysInstallElevated)'

# 18.9.75: WinRM: Disallow unencrypted traffic = 0 (Disabled = encryption required)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WinRM\Client' `
    -Name  'AllowUnencryptedTraffic' `
    -Value 0 `
    -Description '(CIS 18.9.75 - WinRM client encryption required)'

Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WinRM\Service' `
    -Name  'AllowUnencryptedTraffic' `
    -Value 0 `
    -Description '(CIS 18.9.75 - WinRM service encryption required)'

# Enable Windows Defender real-time protection (basic check/set)
Set-RegistryValue `
    -Path  'HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection' `
    -Name  'DisableRealtimeMonitoring' `
    -Value 0 `
    -Description '(CIS - Enable Defender real-time monitoring)'

Write-Status "Windows security registry settings remediation complete."
