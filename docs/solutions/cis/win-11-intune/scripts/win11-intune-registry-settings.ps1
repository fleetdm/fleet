#Requires -RunAsAdministrator
<#
.SYNOPSIS
    CIS Windows 11 Enterprise Benchmark (Intune) v8.1 - Registry-based Security Settings

.DESCRIPTION
    Configures registry-based security settings checked by registry osquery queries
    in the CIS Windows 11 Enterprise (Intune) benchmark. Settings are grouped by
    CIS category.

    This script is idempotent - safe to run multiple times.

.NOTES
    Requires: Administrator privileges
    Run as: SYSTEM or local Administrator
    Settings configured via registry apply immediately unless otherwise noted.
#>

[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Set-RegValue {
    param(
        [string]$Path,
        [string]$Name,
        $Value,
        [string]$Type = 'DWord'
    )
    if (-not (Test-Path $Path)) {
        New-Item -Path $Path -Force | Out-Null
    }
    Set-ItemProperty -Path $Path -Name $Name -Value $Value -Type $Type -Force
    Write-Output "Set [$Path] $Name = $Value"
}

Write-Output "Starting CIS Windows 11 Intune registry remediation..."

# ============================================================
# Section 4 - Control Panel / Account Lockout
# ============================================================

# 4.4.1 Apply UAC restrictions to local accounts on network logons
Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name 'LocalAccountTokenFilterPolicy' -Value 0

# ============================================================
# Section 5 - System / Windows Update
# ============================================================

# 5.1 Ensure automatic updates are enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU' `
    -Name 'NoAutoUpdate' -Value 0

# ============================================================
# Section 9 - Microsoft Edge / Browser settings
# ============================================================

# Disable Internet Explorer as standalone browser (IE mode in Edge only)
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Internet Explorer\Main' `
    -Name 'NotifyDisableIEOptions' -Value 0

# ============================================================
# Section 18 - Administrative Templates: Computer
# ============================================================

# 18.1.1 Ensure 'Prevent enabling lock screen camera' is Enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Personalization' `
    -Name 'NoLockScreenCamera' -Value 1

# 18.1.2 Ensure 'Prevent enabling lock screen slide show' is Enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Personalization' `
    -Name 'NoLockScreenSlideshow' -Value 1

# 18.4.1 Ensure 'MSS: Enable Safe DLL search mode' is Enabled
Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager' `
    -Name 'SafeDllSearchMode' -Value 1

# 18.4.2 Ensure 'MSS: Enable Structured Exception Handling Overwrite Protection (SEHOP)' is Enabled
Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\kernel' `
    -Name 'DisableExceptionChainValidation' -Value 0

# 18.4.3 Ensure 'MSS: IP source routing protection level' is Highest Protection
Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters' `
    -Name 'DisableIPSourceRouting' -Value 2

# 18.4.4 Ensure 'MSS: NetBIOS protection' (NoNameReleaseOnDemand) is Enabled
Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\NetBT\Parameters' `
    -Name 'NoNameReleaseOnDemand' -Value 1

# 18.5 Connectivity
# 18.5.1 Ensure 'Allow Print Spooler to accept client connections' is Disabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\Printers' `
    -Name 'RegisterSpoolerRemoteRpcEndPoint' -Value 2

# 18.5.2 Ensure 'Turn off Microsoft Peer-to-Peer Networking Services' is Enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Peernet' `
    -Name 'Disabled' -Value 1

# 18.8.4 Ensure 'Configure Windows SmartScreen' is Enabled (Warn + prevent bypass)
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' `
    -Name 'EnableSmartScreen' -Value 1
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' `
    -Name 'ShellSmartScreenLevel' -Value 'Block' -Type String

# 18.9.4 Ensure 'Block user from showing account details on sign-in' is Enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' `
    -Name 'BlockUserFromShowingAccountDetailsOnSignin' -Value 1

# 18.9.5 Ensure 'Do not display network selection UI' is Enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' `
    -Name 'DontDisplayNetworkSelectionUI' -Value 1

# 18.9.6 Ensure 'Do not enumerate connected users on domain-joined computers' is Enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' `
    -Name 'DontEnumerateConnectedUsers' -Value 1

# 18.9.7 Ensure 'Enumerate local users on domain-joined computers' is Disabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\System' `
    -Name 'EnumerateLocalUsers' -Value 0

# ============================================================
# Section 18.10 - Autoplay
# ============================================================

# 18.10.1 Ensure 'Disallow Autoplay for non-volume devices' is Enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\Explorer' `
    -Name 'NoAutoplayfornonVolume' -Value 1

# 18.10.2 Ensure 'Set the default behavior for AutoRun' is 'Do not execute any autorun commands'
Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\Explorer' `
    -Name 'NoAutorun' -Value 1

# 18.10.3 Ensure 'Turn off Autoplay' is Enabled (All Drives)
Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\Explorer' `
    -Name 'NoDriveTypeAutoRun' -Value 255

# ============================================================
# Section 18.11 - BitLocker
# ============================================================

# Configure BitLocker startup PIN (informational - requires manual configuration)
Write-Output "INFO: BitLocker startup PIN and encryption settings require manual configuration via BitLocker Drive Encryption control panel or Intune policy."

# ============================================================
# Section 18.17 - SMB Settings
# ============================================================

# Ensure SMBv1 is disabled
Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters' `
    -Name 'SMB1' -Value 0

# Ensure SMBv2 is enabled
Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters' `
    -Name 'SMB2' -Value 1

# Require SMB packet signing (server side)
Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters' `
    -Name 'RequireSecuritySignature' -Value 1
Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters' `
    -Name 'EnableSecuritySignature' -Value 1

# Require SMB packet signing (client/workstation side)
Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters' `
    -Name 'RequireSecuritySignature' -Value 1
Set-RegValue -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters' `
    -Name 'EnableSecuritySignature' -Value 1

# ============================================================
# Section 18.21 - UAC
# ============================================================

# 18.21.1 Ensure 'User Account Control: Admin Approval Mode for the Built-in Administrator account' is Enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name 'FilterAdministratorToken' -Value 1

# 18.21.2 Ensure UAC prompt for administrators: Prompt for consent on the secure desktop
Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name 'ConsentPromptBehaviorAdmin' -Value 2

# 18.21.3 Ensure UAC prompt for standard users: Automatically deny elevation requests
Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name 'ConsentPromptBehaviorUser' -Value 0

# 18.21.4 Ensure 'User Account Control: Detect application installations and prompt for elevation' is Enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name 'EnableInstallerDetection' -Value 1

# 18.21.5 Ensure UAC is Enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name 'EnableLUA' -Value 1

# 18.21.6 Ensure 'User Account Control: Virtualize file and registry write failures to per-user locations' is Enabled
Set-RegValue -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' `
    -Name 'EnableVirtualization' -Value 1

# ============================================================
# Section 19 - Administrative Templates: User
# ============================================================

# 19.1.1 Ensure 'Turn off toast notifications on the lock screen' is Enabled (HKCU)
# Note: Per-user settings (HKCU) cannot be reliably set from a SYSTEM-context script.
# Configure this via Intune device configuration profile: Experience/AllowToastNotifications = 0
Write-Output "INFO: Toast notification lock screen settings should be configured via Intune device configuration profile (Experience/AllowToastNotifications)"

Write-Output ""
Write-Output "CIS registry remediation complete."
Write-Output "Restart may be required for some settings to take effect."
