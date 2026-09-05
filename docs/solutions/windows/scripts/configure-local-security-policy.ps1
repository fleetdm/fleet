# configure-local-security-policy.ps1
# CIS Windows 11 Enterprise Benchmark v4.0.0 - Section 2.3 Local Security Policy
# Configures security options that require direct registry or secedit manipulation

$ErrorActionPreference = 'Stop'

# 2.3.2.1 - Force audit policy subcategory settings = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'SCENoApplyLegacyAuditPolicy' -Value 1 -Type DWord -Force

# 2.3.2.2 - Shut down system if unable to log security audits = Disabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'CrashOnAuditFail' -Value 0 -Type DWord -Force

# 2.3.4.1 (L2) - Prevent users from installing printer drivers = Enabled
$printerPath = 'HKLM:\SYSTEM\CurrentControlSet\Control\Print\Providers\LanMan Print Services\Servers'
if (-not (Test-Path $printerPath)) { New-Item -Path $printerPath -Force | Out-Null }
Set-ItemProperty -Path $printerPath -Name 'AddPrinterDrivers' -Value 1 -Type DWord -Force

# 2.3.6.1 - Digitally encrypt or sign secure channel data (always) = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters' -Name 'RequireSignOrSeal' -Value 1 -Type DWord -Force

# 2.3.6.2 - Digitally encrypt secure channel data (when possible) = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters' -Name 'SealSecureChannel' -Value 1 -Type DWord -Force

# 2.3.6.3 - Digitally sign secure channel data (when possible) = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters' -Name 'SignSecureChannel' -Value 1 -Type DWord -Force

# 2.3.6.4 - Disable machine account password changes = Disabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters' -Name 'DisablePasswordChange' -Value 0 -Type DWord -Force

# 2.3.6.5 - Maximum machine account password age = 30 days
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters' -Name 'MaximumPasswordAge' -Value 30 -Type DWord -Force

# 2.3.6.6 - Require strong session key = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\Netlogon\Parameters' -Name 'RequireStrongKey' -Value 1 -Type DWord -Force

# 2.3.7.1 - Do not require CTRL+ALT+DEL = Disabled (require it)
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'DisableCAD' -Value 0 -Type DWord -Force

# 2.3.7.3 (BL) - Machine account lockout threshold = 10
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'MaxDevicePasswordFailedAttempts' -Value 10 -Type DWord -Force

# 2.3.7.5 - Message text for logon
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'LegalNoticeText' -Value 'This system is for authorized use only. By using this system, you consent to monitoring.' -Type String -Force

# 2.3.7.6 - Message title for logon
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'LegalNoticeCaption' -Value 'NOTICE: Authorized Use Only' -Type String -Force

# 2.3.7.7 (L2) - Cache logons = 4
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon' -Name 'CachedLogonsCount' -Value '4' -Type String -Force

# 2.3.7.8 - Prompt password change before expiration = 14 days
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon' -Name 'PasswordExpiryWarning' -Value 14 -Type DWord -Force

# 2.3.7.9 - Smart card removal behavior = Lock Workstation (1)
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon' -Name 'ScRemoveOption' -Value '1' -Type String -Force

# 2.3.8.1 - SMB client digitally sign (always) = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters' -Name 'RequireSecuritySignature' -Value 1 -Type DWord -Force

# 2.3.8.2 - SMB client digitally sign (if server agrees) = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters' -Name 'EnableSecuritySignature' -Value 1 -Type DWord -Force

# 2.3.8.3 - Send unencrypted password to SMB servers = Disabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanmanWorkstation\Parameters' -Name 'EnablePlainTextPassword' -Value 0 -Type DWord -Force

# 2.3.9.1 - Idle time before suspending session = 15 minutes
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters' -Name 'AutoDisconnect' -Value 15 -Type DWord -Force

# 2.3.9.2 - SMB server digitally sign (always) = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters' -Name 'RequireSecuritySignature' -Value 1 -Type DWord -Force

# 2.3.9.3 - SMB server digitally sign (if client agrees) = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters' -Name 'EnableSecuritySignature' -Value 1 -Type DWord -Force

# 2.3.9.4 - Disconnect clients when logon hours expire = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters' -Name 'EnableForcedLogOff' -Value 1 -Type DWord -Force

# 2.3.9.5 - Server SPN target name validation level = Accept if provided (1)
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters' -Name 'SMBServerNameHardeningLevel' -Value 1 -Type DWord -Force

# 2.3.10.1 - Allow anonymous SID/Name translation = Disabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'TurnOffAnonymousBlock' -Value 1 -Type DWord -Force

# 2.3.10.2 - Do not allow anonymous enumeration of SAM accounts = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'RestrictAnonymousSAM' -Value 1 -Type DWord -Force

# 2.3.10.3 - Do not allow anonymous enumeration of SAM accounts and shares = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'RestrictAnonymous' -Value 1 -Type DWord -Force

# 2.3.10.4 - Do not allow storage of passwords for network authentication = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'DisableDomainCreds' -Value 1 -Type DWord -Force

# 2.3.10.5 - Let Everyone permissions apply to anonymous users = Disabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'EveryoneIncludesAnonymous' -Value 0 -Type DWord -Force

# 2.3.10.6 - Named Pipes accessible anonymously = None
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters' -Name 'NullSessionPipes' -Value @() -Type MultiString -Force

# 2.3.10.9 - Restrict anonymous access to Named Pipes and Shares = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters' -Name 'RestrictNullSessAccess' -Value 1 -Type DWord -Force

# 2.3.10.10 - Restrict clients allowed to make remote calls to SAM = Admins: Remote Access: Allow
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'RestrictRemoteSAM' -Value 'O:BAG:BAD:(A;;RC;;;BA)' -Type String -Force

# 2.3.10.11 - Shares accessible anonymously = None
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LanManServer\Parameters' -Name 'NullSessionShares' -Value @() -Type MultiString -Force

# 2.3.10.12 - Sharing and security model = Classic (0)
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'ForceGuest' -Value 0 -Type DWord -Force

# 2.3.11.1 - Allow Local System to use computer identity for NTLM = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'UseMachineId' -Value 1 -Type DWord -Force

# 2.3.11.2 - Allow LocalSystem NULL session fallback = Disabled
$msvPath = 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa\MSV1_0'
if (-not (Test-Path $msvPath)) { New-Item -Path $msvPath -Force | Out-Null }
Set-ItemProperty -Path $msvPath -Name 'AllowNullSessionFallback' -Value 0 -Type DWord -Force

# 2.3.11.3 - Allow PKU2U authentication = Disabled
$pku2uPath = 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa\pku2u'
if (-not (Test-Path $pku2uPath)) { New-Item -Path $pku2uPath -Force | Out-Null }
Set-ItemProperty -Path $pku2uPath -Name 'AllowOnlineID' -Value 0 -Type DWord -Force

# 2.3.11.4 - Kerberos encryption types = AES128+AES256+Future (2147483640)
$kerbPath = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System\Kerberos\Parameters'
if (-not (Test-Path $kerbPath)) { New-Item -Path $kerbPath -Force | Out-Null }
Set-ItemProperty -Path $kerbPath -Name 'SupportedEncryptionTypes' -Value 2147483640 -Type DWord -Force

# 2.3.11.5 - Do not store LAN Manager hash = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Lsa' -Name 'NoLMHash' -Value 1 -Type DWord -Force

# 2.3.11.8 - LDAP client encryption requirements = Negotiate sealing (1)
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\LDAP' -Name 'LDAPClientIntegrity' -Value 1 -Type DWord -Force

# 2.3.11.10 - Minimum session security for NTLM SSP clients = NTLMv2 + 128-bit (537395200)
Set-ItemProperty -Path $msvPath -Name 'NtlmMinClientSec' -Value 537395200 -Type DWord -Force

# 2.3.11.11 - Minimum session security for NTLM SSP servers = NTLMv2 + 128-bit (537395200)
Set-ItemProperty -Path $msvPath -Name 'NtlmMinServerSec' -Value 537395200 -Type DWord -Force

# 2.3.11.12 - Restrict NTLM: Audit Incoming Traffic = Enable auditing for all accounts (2)
Set-ItemProperty -Path $msvPath -Name 'AuditReceivingNTLMTraffic' -Value 2 -Type DWord -Force

# 2.3.11.13 - Restrict NTLM: Outgoing traffic = Audit all (1)
Set-ItemProperty -Path $msvPath -Name 'RestrictSendingNTLMTraffic' -Value 1 -Type DWord -Force

# 2.3.14.1 (L2) - Force strong key protection = User prompted (1)
$cryptPath = 'HKLM:\SOFTWARE\Policies\Microsoft\Cryptography'
if (-not (Test-Path $cryptPath)) { New-Item -Path $cryptPath -Force | Out-Null }
Set-ItemProperty -Path $cryptPath -Name 'ForceKeyProtection' -Value 1 -Type DWord -Force

# 2.3.15.1 - Require case insensitivity for non-Windows subsystems = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Kernel' -Name 'ObCaseInsensitive' -Value 1 -Type DWord -Force

# 2.3.15.2 - Strengthen default permissions of internal system objects = Enabled
Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager' -Name 'ProtectionMode' -Value 1 -Type DWord -Force

# 2.3.17.1 - UAC: Admin Approval Mode for Built-in Admin = Enabled
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'FilterAdministratorToken' -Value 1 -Type DWord -Force

# 2.3.17.2 - UAC: Behavior of elevation prompt for admins = Prompt for consent on secure desktop (2)
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'ConsentPromptBehaviorAdmin' -Value 2 -Type DWord -Force

# 2.3.17.3 - UAC: Behavior of elevation prompt for standard users = Auto deny (0)
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'ConsentPromptBehaviorUser' -Value 0 -Type DWord -Force

# 2.3.17.4 - UAC: Detect application installations = Enabled
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'EnableInstallerDetection' -Value 1 -Type DWord -Force

# 2.3.17.5 - UAC: Only elevate UIAccess in secure locations = Enabled
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'EnableSecureUIAPaths' -Value 1 -Type DWord -Force

# 2.3.17.6 - UAC: Run all administrators in Admin Approval Mode = Enabled
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'EnableLUA' -Value 1 -Type DWord -Force

# 2.3.17.7 - UAC: Switch to secure desktop = Enabled
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'PromptOnSecureDesktop' -Value 1 -Type DWord -Force

# 2.3.17.8 - UAC: Virtualize file and registry write failures = Enabled
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'EnableVirtualization' -Value 1 -Type DWord -Force

# Rename Administrator and Guest accounts via secedit (2.3.1.3, 2.3.1.4)
$infContent = @"
[Unicode]
Unicode=yes
[System Access]
NewAdministratorName = "LocalAdmin"
NewGuestName = "LocalGuest"
[Version]
signature="`$CHICAGO`$"
Revision=1
"@

$infPath = "$env:TEMP\fleet-secpol.inf"
$dbPath = "$env:TEMP\fleet-secpol.sdb"
$infContent | Out-File -FilePath $infPath -Encoding Unicode -Force
secedit /configure /db $dbPath /cfg $infPath /areas SECURITYPOLICY /quiet
Remove-Item -Path $infPath, $dbPath -Force -ErrorAction SilentlyContinue

Write-Output "Local Security Policy configuration complete."
