package main

import "strings"

// VettedQuery represents a 100% source-verified, production-safe osquery policy query.
// ALL queries in this file are sourced verbatim from:
//   - macOS:   https://github.com/karmine05/fleet_policies/blob/main/CIS-8.1/macOS26/cis-macOSTahoe-policies.yaml
//   - Linux:   https://github.com/karmine05/fleet_policies/blob/main/CIS-8.1/ubuntu24/cis-ubuntu24-server-policies.yaml
//   - Win L1:  https://github.com/karmine05/fleet_policies/blob/main/CIS-8.1/win11/intune/l1_win11_intune.yaml
//   - Win L2:  https://github.com/karmine05/fleet_policies/blob/main/CIS-8.1/win11/intune/l2_win11_intune.yaml
//   - Win BL:  https://github.com/karmine05/fleet_policies/blob/main/CIS-8.1/win11/intune/bl_win11_intune.yaml
//
// DO NOT add any query that has not been read verbatim from these source files.
type VettedQuery struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Platform    string `json:"platform"` // "darwin", "windows", or "linux"
	Category    string `json:"category"` // e.g. "CIS"
	Level       string `json:"level"`    // "L1", "L2", "BL"
	CISRef      string `json:"cis_ref"`  // CIS safeguard reference
	SQL         string `json:"sql"`
}

// vettedQueryLibrary contains queries sourced verbatim from the above repos.
var vettedQueryLibrary = []VettedQuery{

	// =========================================================================
	// MACOS (DARWIN) — CIS-8.1 Tahoe
	// Source: karmine05/fleet_policies — CIS-8.1/macOS26/cis-macOSTahoe-policies.yaml
	// =========================================================================
	{
		Name:        "CIS 1.1 (L1) Ensure Apple-provided Software Updates Are Installed",
		Description: "Software vendors release security patches and software updates for their products when security vulnerabilities are discovered.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS1.1",
		SQL:         `SELECT 1 FROM software_update WHERE software_update_required = '0';`,
	},
	{
		Name:        "CIS 6.3.1 (L1) Ensure Automatic Opening of Safe Files in Safari Is Disabled",
		Description: "Disables automatic opening of so-called 'safe' files after download to prevent unintended execution.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS6.3.1",
		SQL: `SELECT 1 WHERE 
  EXISTS (
    SELECT 1 FROM managed_policies WHERE 
        domain='com.apple.Safari' AND 
        name='AutoOpenSafeDownloads' AND 
        (value = 0 OR value = 'false')
    )
  AND NOT EXISTS (
    SELECT 1 FROM managed_policies WHERE 
        domain='com.apple.Safari' AND 
        name='AutoOpenSafeDownloads' AND 
        (value != 0 AND value != 'false')
    );`,
	},
	{
		Name:        "CIS 6.3.3 (L1) Ensure Warn When Visiting A Fraudulent Website in Safari Is Enabled",
		Description: "Ensures phishing/malware warning feature is active to alert users of known fraudulent sites.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS6.3.3",
		SQL: `SELECT 1 WHERE 
  EXISTS (
    SELECT 1 FROM managed_policies
    WHERE domain='com.apple.Safari'
      AND name='WarnAboutFraudulentWebsites'
      AND (value=1 OR value='true')
  )
  AND NOT EXISTS (
    SELECT 1 FROM managed_policies
    WHERE domain='com.apple.Safari'
      AND name='WarnAboutFraudulentWebsites'
      AND (value!=1 AND value!='true')
  );`,
	},
	{
		Name:        "CIS 6.3.7 (L1) Ensure Show Full Website Address in Safari Is Enabled",
		Description: "Displays full URL in address bar improving user awareness of actual destination host/path.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS6.3.7",
		SQL: `SELECT 1 WHERE 
  EXISTS (
    SELECT 1 FROM managed_policies WHERE 
        domain='com.apple.Safari' AND 
        name='ShowFullURLInSmartSearchField' AND 
        (value = 1 OR value = 'true') 
    )
  AND NOT EXISTS (
    SELECT 1 FROM managed_policies WHERE 
        domain='com.apple.Safari' AND 
        name='ShowFullURLInSmartSearchField' AND 
        (value != 1 AND value != 'true')
    );`,
	},
	{
		Name:        "CIS 6.3.10 (L1) Ensure Show Status Bar Is Enabled in Safari",
		Description: "Shows status bar revealing target link destinations aiding phishing detection.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS6.3.10",
		SQL: `SELECT 1 WHERE 
  EXISTS (
    SELECT 1 FROM managed_policies
    WHERE domain='com.apple.Safari'
      AND name='ShowOverlayStatusBar'
      AND (value=1 OR value='true')
  )
  AND NOT EXISTS (
    SELECT 1 FROM managed_policies
    WHERE domain='com.apple.Safari'
      AND name='ShowOverlayStatusBar'
      AND (value!=1 AND value!='true')
  );`,
	},
	{
		Name:        "CIS 6.4.1 (L1) Ensure Secure Keyboard Entry Terminal.app Is Enabled",
		Description: "Secure Keyboard Entry reduces risk of keystroke interception by other processes in Terminal.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS6.4.1",
		SQL: `SELECT 1 WHERE 
  EXISTS (
    SELECT 1 FROM managed_policies WHERE 
        domain='com.apple.Terminal' AND 
        name='SecureKeyboardEntry' AND 
        (value = 1 OR value = 'true')
    )
  AND NOT EXISTS (
    SELECT 1 FROM managed_policies WHERE 
        domain='com.apple.Terminal' AND 
        name='SecureKeyboardEntry' AND 
        (value != 1 AND value != 'true')
    );`,
	},
	{
		Name:        "CIS 5.1.2 (L1) Ensure System Integrity Protection (SIP) Is Enabled",
		Description: "Confirms SIP is active preventing modification of protected system locations.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS5.1.2",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT 1 FROM sip_config WHERE config_flag='sip' AND enabled=1);`,
	},
	{
		Name:        "CIS 5.1.4 (L1) Ensure Signed System Volume (SSV) Is Enabled",
		Description: "Confirms SSV cryptographic seal integrity ensuring system volume tamper detection.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS5.1.4",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT 1 FROM csrutil_info WHERE ssv_enabled='1');`,
	},
	{
		Name:        "CIS 5.2.1 (L1) Ensure Password Account Lockout Threshold Is Configured",
		Description: "Enforces lockout after a defined number of failed authentication attempts.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS5.2.1",
		SQL:         `SELECT 1 FROM pwd_policy where max_failed_attempts <= 5;`,
	},
	{
		Name:        "CIS 5.2.2 (L1) Ensure Password Minimum Length Is Configured",
		Description: "Ensures a minimum password length of 15 or more characters.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS5.2.2",
		SQL: `SELECT 1 
FROM (
SELECT cast(lengthtxt as integer(2)) minlength 
FROM (
SELECT SUBSTRING(length, 1, 2) AS lengthtxt 
FROM (
SELECT policy_description, policy_identifier, split(policy_content, '{', 1) AS length 
FROM password_policy 
WHERE policy_identifier LIKE '%minLength')) 
WHERE minlength >= 15);`,
	},
	{
		Name:        "CIS 5.2.8 (L1) Ensure Password History Is Set to At Least 24",
		Description: "Prevents reuse of recently used passwords (history >= 24).",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS5.2.8",
		SQL:         `SELECT 1 FROM pwd_policy WHERE history_depth >= 24;`,
	},
	{
		Name:        "CIS 5.3.1 (L1) Ensure All User APFS Volumes Are Encrypted",
		Description: "Confirms all mounted user data APFS volumes have encryption enabled.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS5.3.1",
		SQL: `SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM apfs_volumes
  WHERE role NOT IN ('VM','Update','Recovery','Preboot','xART','Hardware')
    AND filevault != 1
);`,
	},
	{
		Name:        "CIS 5.4 (L1) Ensure the Sudo Timeout Period Is Set to Zero",
		Description: "Requires re-authentication for every sudo invocation by setting timeout to 0.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS5.4",
		SQL: `SELECT 1 WHERE EXISTS(
  SELECT * FROM file WHERE path = '/etc/sudoers.d' AND uid = 0 AND gid = 0
) AND EXISTS(
  SELECT
    COALESCE(JSON_EXTRACT(
      json_result, '$.Authentication timestamp timeout'
    ), '') AS authentication_timestamp_timeout
  FROM sudo_info WHERE authentication_timestamp_timeout = '0.0 minutes'
);`,
	},
	{
		Name:        "CIS 5.6 (L1) Ensure the root Account Is Disabled",
		Description: "Ensures the root user cannot directly authenticate.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS5.6",
		SQL:         `SELECT 1 from dscl WHERE command = 'read' AND path = '/Users/root' AND key = 'AuthenticationAuthority' AND value = '';`,
	},
	{
		Name:        "CIS 5.9 (L1) Ensure the Guest Home Folder Does Not Exist",
		Description: "Validates removal of residual Guest user home directory.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS5.9",
		SQL:         `SELECT 1 WHERE NOT EXISTS (SELECT * FROM file WHERE path = '/Users/Guest');`,
	},
	{
		Name:        "CIS 4.2 (L1) Ensure HTTP Server Is Disabled",
		Description: "Ensures built-in Apache httpd is not running reducing attack surface.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.2",
		SQL:         `SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE path='/usr/sbin/httpd');`,
	},
	{
		Name:        "CIS 4.3 (L1) Ensure NFS Server Is Disabled",
		Description: "Disables legacy NFS file sharing components to minimize exposure.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.3",
		SQL: `SELECT 1 WHERE (
  NOT EXISTS(SELECT 1 FROM processes WHERE path='/sbin/nfsd')
  AND NOT EXISTS(SELECT 1 FROM file WHERE path='/etc/exports')
);`,
	},
	{
		Name:        "CIS 2.6.4 (L1) Ensure Limit Ad Tracking Is Enabled",
		Description: "Disables Apple personalized advertising to reduce privacy exposure.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS2.6.4",
		SQL: `SELECT 1 WHERE (
  EXISTS (SELECT 1 FROM managed_policies WHERE domain='com.apple.applicationaccess' AND name='allowApplePersonalizedAdvertising' AND (value=0 OR value='false'))
  AND NOT EXISTS (SELECT 1 FROM managed_policies WHERE domain='com.apple.applicationaccess' AND name='allowApplePersonalizedAdvertising' AND (value!=0 AND value!='false'))
);`,
	},
	{
		Name:        "CIS 5.1.1 (L1) Ensure Home Folders Are Secure",
		Description: "Validates user home directories are not group/other readable beyond minimal execute traversal.",
		Platform:    "darwin",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS5.1.1",
		SQL: `SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM file WHERE path LIKE '/Users/%' AND path != '/Users/Shared/' AND mode NOT IN ('0700','0701','0710','0711')
);`,
	},

	// =========================================================================
	// LINUX — CIS-8.1 Ubuntu 24
	// Source: karmine05/fleet_policies — CIS-8.1/ubuntu24/cis-ubuntu24-server-policies.yaml
	// =========================================================================
	{
		Name:        "CIS Linux - SSH MaxAuthTries Is 3",
		Description: "Enforces SSH MaxAuthTries to 3 reducing brute force attack surface.",
		Platform:    "linux",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.6",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT * FROM augeas WHERE path='/files/etc/ssh/sshd_config/MaxAuthTries' AND LOWER(value)=LOWER('3'));`,
	},
	{
		Name:        "CIS Linux - SSH LoginGraceTime Is 30",
		Description: "Enforces SSH LoginGraceTime to 30 seconds reducing session abuse risk.",
		Platform:    "linux",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.6",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT * FROM augeas WHERE path='/files/etc/ssh/sshd_config/LoginGraceTime' AND LOWER(value)=LOWER('30'));`,
	},
	{
		Name:        "CIS Linux - SSH X11Forwarding Disabled",
		Description: "Ensures SSH X11Forwarding is disabled to prevent remote GUI session abuse.",
		Platform:    "linux",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.6",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT * FROM augeas WHERE path='/files/etc/ssh/sshd_config/X11Forwarding' AND LOWER(value)=LOWER('no'));`,
	},
	{
		Name:        "CIS Linux - SSH AllowTcpForwarding Disabled",
		Description: "Ensures SSH AllowTcpForwarding is disabled to reduce tunneling risk.",
		Platform:    "linux",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.6",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT * FROM augeas WHERE path='/files/etc/ssh/sshd_config/AllowTcpForwarding' AND LOWER(value)=LOWER('no'));`,
	},
	{
		Name:        "CIS Linux - SSH Idle Session Timeout Enforced (300s)",
		Description: "Enforces SSH ClientAliveInterval=300 for idle session termination.",
		Platform:    "linux",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.6",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT * FROM augeas WHERE path='/files/etc/ssh/sshd_config/ClientAliveInterval' AND LOWER(value)=LOWER('300'));`,
	},
	{
		Name:        "CIS Linux - SSH PermitEmptyPasswords Disabled",
		Description: "Ensures empty passwords are not permitted via SSH.",
		Platform:    "linux",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.6",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT * FROM augeas WHERE path='/files/etc/ssh/sshd_config/PermitEmptyPasswords' AND LOWER(value)=LOWER('no'));`,
	},
	{
		Name:        "CIS Linux - SSH IgnoreRhosts Enabled",
		Description: "Ensures SSH ignores .rhosts files for host-based authentication.",
		Platform:    "linux",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.6",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT * FROM augeas WHERE path='/files/etc/ssh/sshd_config/IgnoreRhosts' AND LOWER(value)=LOWER('yes'));`,
	},
	{
		Name:        "CIS Linux - SSH HostbasedAuthentication Disabled",
		Description: "Ensures SSH host-based authentication is disabled.",
		Platform:    "linux",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.6",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT * FROM augeas WHERE path='/files/etc/ssh/sshd_config/HostbasedAuthentication' AND LOWER(value)=LOWER('no'));`,
	},
	{
		Name:        "CIS Linux - Disk Encryption Configured (crypttab Entries Present)",
		Description: "Verifies presence of active dm-crypt/LUKS mappings declared in /etc/crypttab.",
		Platform:    "linux",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS3.6",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT * FROM file_lines WHERE path='/etc/crypttab' AND line NOT LIKE '#%' AND LENGTH(TRIM(line)) > 0);`,
	},
	{
		Name:        "CIS Linux - Backup Tool Installed (restic or borgbackup)",
		Description: "Ensures a modern backup utility (restic or borgbackup) is installed.",
		Platform:    "linux",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS10.1",
		SQL:         `SELECT 1 WHERE EXISTS (SELECT * FROM deb_packages WHERE name IN ('restic','borgbackup'));`,
	},
	{
		Name:        "CIS Linux - Crash Telemetry (apport) Disabled",
		Description: "Confirms apport crash reporting is disabled to reduce unsolicited data transmission.",
		Platform:    "linux",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS13.2",
		SQL: `SELECT 1 WHERE EXISTS (
  SELECT * FROM file_lines WHERE path='/etc/default/apport' AND line LIKE 'enabled=0'
);`,
	},

	// =========================================================================
	// WINDOWS — CIS-8.1 Win11 Intune (L1)
	// Source: https://github.com/karmine05/fleet_policies/blob/main/CIS-8.1/win11/intune/l1_win11_intune.yaml
	// =========================================================================
	{
		Name:        "CIS 1.1 (L1) Ensure 'Allow Cortana Above Lock' is set to 'Block' (Automated)",
		Description: "This policy setting determines whether or not the user can interact with Cortana using speech while the system is locked. The recommended state for this setting is: Block.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS1.1",
		SQL:         `SELECT 1 FROM mdm_bridge WHERE mdm_command_input = '<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Result/AboveLock/AllowCortanaAboveLock</LocURI></Target></Item></Get></SyncBody>' AND mdm_command_output = '0';`,
	},
	{
		Name:        "CIS 4.4.1 (L1) Ensure 'Apply UAC restrictions to local accounts on network logons' is set to 'Enabled' (Automated)",
		Description: "This setting controls whether local accounts can be used for remote administration via network logon. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.4.1",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System\LocalAccountTokenFilterPolicy' AND data = '0';`,
	},
	{
		Name:        "CIS 4.4.2 (L1) Ensure 'Configure SMB v1 client driver' is set to 'Enabled: Disable driver (recommended)' (Automated)",
		Description: "This setting configures the start type for the SMBv1 client driver. The recommended state for this setting is: Enabled: Disable driver (recommended).",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.4.2",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\mrxsmb10\Start' AND data = '4';`,
	},
	{
		Name:        "CIS 4.4.3 (L1) Ensure 'Configure SMB v1 server' is set to 'Disabled' (Automated)",
		Description: "This setting configures the server-side processing of SMBv1. The recommended state for this setting is: Disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.4.3",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\LanmanServer\Parameters\SMB1' AND data = '0';`,
	},
	{
		Name:        "CIS 4.4.4 (L1) Ensure 'Enable Structured Exception Handling Overwrite Protection (SEHOP)' is set to 'Enabled' (Automated)",
		Description: "Windows includes support for SEHOP. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.4.4",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Session Manager\kernel\DisableExceptionChainValidation' AND data = '0';`,
	},
	{
		Name:        "CIS 4.4.5 (L1) Ensure 'WDigest Authentication' is set to 'Disabled' (Automated)",
		Description: "When WDigest authentication is enabled, Lsass.exe retains a copy of the user's plaintext password in memory. The recommended state for this setting is: Disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.4.5",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\SecurityProviders\WDigest\UseLogonCredential' AND data = '0';`,
	},
	{
		Name:        "CIS 4.5.2 (L1) Ensure 'MSS: (DisableIPSourceRouting IPv6)' is set to 'Enabled: Highest protection' (Automated)",
		Description: "IP source routing is a mechanism that allows the sender to determine the IP route for a datagram. The recommended state: Enabled: Highest protection, source routing is completely disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.5.2",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Tcpip6\Parameters\DisableIPSourceRouting' AND data = '2';`,
	},
	{
		Name:        "CIS 4.5.3 (L1) Ensure 'MSS: (DisableIPSourceRouting)' is set to 'Enabled: Highest protection' (Automated)",
		Description: "IP source routing protection level for IPv4. The recommended state: Enabled: Highest protection, source routing is completely disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.5.3",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\DisableIPSourceRouting' AND data = '2';`,
	},
	{
		Name:        "CIS 4.5.5 (L1) Ensure 'MSS: (EnableICMPRedirect) Allow ICMP redirects to override OSPF generated routes' is set to 'Disabled' (Automated)",
		Description: "ICMP redirects cause the IPv4 stack to plumb host routes overriding OSPF-generated routes. The recommended state for this setting is: Disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.5.5",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\EnableICMPRedirect' AND data = '0';`,
	},
	{
		Name:        "CIS 4.5.7 (L1) Ensure 'MSS: (NoNameReleaseOnDemand) Allow the computer to ignore NetBIOS name release requests except from WINS servers' is set to 'Enabled' (Automated)",
		Description: "This setting determines whether the computer releases its NetBIOS name when it receives a name-release request. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.5.7",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\NetBT\Parameters\NoNameReleaseOnDemand' AND data = '1';`,
	},
	{
		Name:        "CIS 4.5.9 (L1) Ensure 'MSS: (SafeDllSearchMode) Enable Safe DLL search mode (recommended)' is set to 'Enabled' (Automated)",
		Description: "Safe DLL search mode forces the system to search the system path first before the working directory. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.5.9",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Session Manager\SafeDllSearchMode' AND data = '1';`,
	},
	{
		Name:        "CIS 4.5.10 (L1) Ensure 'MSS: (ScreenSaverGracePeriod)' is set to 'Enabled: 5 or fewer seconds' (Automated)",
		Description: "Windows includes a grace period between when the screen saver is launched and when the console is actually locked. The recommended state for this setting is: Enabled: 5 or fewer seconds.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.5.10",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon\ScreenSaverGracePeriod' AND CAST(data AS INTEGER) <= 5;`,
	},
	{
		Name:        "CIS 4.5.13 (L1) Ensure 'MSS: (WarningLevel) Percentage threshold for the security event log' is set to 'Enabled: 90% or less' (Automated)",
		Description: "This setting can generate a security audit in the Security event log when the log reaches a user-defined threshold. The recommended state for this setting is: Enabled: 90% or less.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.5.13",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Eventlog\Security\WarningLevel' AND CAST(data AS INTEGER) <= 90;`,
	},
	{
		Name:        "CIS 4.6.4.1 (L1) Ensure 'Turn off multicast name resolution' is set to 'Enabled' (Automated)",
		Description: "LLMNR is a secondary name resolution protocol. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.6.4.1",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows NT\DNSClient\EnableMulticast' AND data = '0';`,
	},
	{
		Name:        "CIS 4.6.18.1 (L1) Ensure 'Minimize the number of simultaneous connections to the Internet or a Windows Domain' is set to 'Enabled: 3 = Prevent Wi-Fi when on Ethernet' (Automated)",
		Description: "This policy setting prevents computers from establishing multiple simultaneous connections to either the Internet or to a Windows domain. The recommended state for this setting is: Enabled: 3 = Prevent Wi-Fi when on Ethernet.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.6.18.1",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\WcmSvc\GroupPolicy\fMinimizeConnections' AND data = '3';`,
	},
	{
		Name:        "CIS 4.7.1 (L1) Ensure 'Allow Print Spooler to accept client connections' is set to 'Disabled' (Automated)",
		Description: "This policy setting controls whether the Print Spooler service will accept client connections. The recommended state for this setting is: Disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.7.1",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows NT\Printers\RegisterSpoolerRemoteRpcEndPoint' AND data = '2';`,
	},
	{
		Name:        "CIS 4.7.8 (L1) Ensure 'Limits print driver installation to Administrators' is set to 'Enabled' (Automated)",
		Description: "This policy setting controls whether users who aren't Administrators can install print drivers on the system. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.7.8",
		SQL:         `SELECT 1 FROM mdm_bridge WHERE mdm_command_input = '<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Result/Printers/RestrictDriverInstallationToAdministrators</LocURI></Target></Item></Get></SyncBody>' AND mdm_command_output LIKE '%<enabled/>%';`,
	},
	{
		Name:        "CIS 4.9.1.1 (L1) Ensure 'Turn off toast notifications on the lock screen (User)' is set to 'Enabled' (Automated)",
		Description: "This policy setting turns off toast notifications on the lock screen. The recommended state for this setting is Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.9.1.1",
		SQL:         `SELECT 1 FROM registry WHERE path LIKE 'HKEY_USERS\S-1-%\SOFTWARE\Policies\Microsoft\Windows\CurrentVersion\PushNotifications\NoToastApplicationNotificationOnLockScreen' AND data = '1';`,
	},
	{
		Name:        "CIS 4.10.4.1 (L1) Ensure 'Include command line in process creation events' is set to 'Enabled' (Automated)",
		Description: "This policy setting controls whether the process creation command line text is logged in security audit events when a new process has been created. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.10.4.1",
		SQL:         `SELECT 1 FROM mdm_bridge WHERE mdm_command_input = '<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Result/ADMX_AuditSettings/IncludeCmdLine</LocURI></Target></Item></Get></SyncBody>' AND mdm_command_output LIKE '%<Enabled/>%';`,
	},
	{
		Name:        "CIS 4.10.5.1 (L1) Ensure 'Encryption Oracle Remediation' is set to 'Enabled: Force Updated Clients' (Automated)",
		Description: "Some versions of the CredSSP protocol are vulnerable to an encryption oracle attack against the client. The recommended state for this setting is: Enabled: Force Updated Clients.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L1",
		CISRef:      "CIS4.10.5.1",
		SQL:         `SELECT 1 FROM mdm_bridge WHERE mdm_command_input = '<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Result/ADMX_CredSsp/AllowEncryptionOracle</LocURI></Target></Item></Get></SyncBody>' AND mdm_command_output LIKE '%<data id="AllowEncryptionOracleDrop" value="0"/>%';`,
	},

	// =========================================================================
	// WINDOWS — CIS-8.1 Win11 Intune (L2)
	// Source: https://github.com/karmine05/fleet_policies/blob/main/CIS-8.1/win11/intune/l2_win11_intune.yaml
	// =========================================================================
	{
		Name:        "CIS 4.5.4 (L2) Ensure 'MSS: (DisableSavePassword) Prevent the dial-up password from being saved (recommended)' is set to 'Enabled' (Automated)",
		Description: "For security, administrators may want to prevent users from caching dial-up passwords. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.5.4",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\RasMan\Parameters\DisableSavePassword' AND data = '1';`,
	},
	{
		Name:        "CIS 4.5.6 (L2) Ensure 'MSS: (KeepAliveTime) How often keep-alive packets are sent in milliseconds' is set to 'Enabled: 300,000 or 5 minutes' (Automated)",
		Description: "This value controls how often TCP attempts to verify that an idle connection is still intact by sending a keep-alive packet. The recommended state for this setting is: Enabled: 300,000 or 5 minutes (recommended).",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.5.6",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\KeepAliveTime' AND CAST(data AS INTEGER) <= 300000;`,
	},
	{
		Name:        "CIS 4.5.8 (L2) Ensure 'MSS: (PerformRouterDiscovery) Allow IRDP to detect and configure Default Gateway addresses' is set to 'Disabled' (Automated)",
		Description: "This setting is used to enable or disable the Internet Router Discovery Protocol (IRDP). The recommended state for this setting is: Disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.5.8",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\PerformRouterDiscovery' AND data = '0';`,
	},
	{
		Name:        "CIS 4.5.11 (L2) Ensure 'MSS: (TcpMaxDataRetransmissions IPv6)' is set to 'Enabled: 3' (Automated)",
		Description: "This setting controls the number of times that TCP retransmits an individual data segment before the connection is aborted (IPv6). The recommended state for this setting is: Enabled: 3.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.5.11",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\TCPIP6\Parameters\TcpMaxDataRetransmissions' AND CAST(data AS INTEGER) <= 3;`,
	},
	{
		Name:        "CIS 4.5.12 (L2) Ensure 'MSS: (TcpMaxDataRetransmissions)' is set to 'Enabled: 3' (Automated)",
		Description: "This setting controls the number of times that TCP retransmits an individual data segment before the connection is aborted (IPv4). The recommended state for this setting is: Enabled: 3.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.5.12",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\TcpMaxDataRetransmissions' AND CAST(data AS INTEGER) <= 3;`,
	},
	{
		Name:        "CIS 4.6.8.1 (L2) Ensure 'Turn on Mapper I/O (LLTDIO) driver' is set to 'Disabled' (Automated)",
		Description: "LLTDIO allows a computer to discover the topology of a network it's connected to. The recommended state for this setting is: Disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.6.8.1",
		SQL:         `SELECT 1 WHERE COALESCE((SELECT UPPER(start_type) FROM services WHERE name = 'LLTDIO'), 'DISABLED') = 'DISABLED';`,
	},
	{
		Name:        "CIS 4.6.8.2 (L2) Ensure 'Turn on Responder (RSPNDR) driver' is set to 'Disabled' (Automated)",
		Description: "The Responder allows a computer to participate in Link Layer Topology Discovery requests. The recommended state for this setting is: Disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.6.8.2",
		SQL:         `SELECT 1 WHERE COALESCE((SELECT UPPER(start_type) FROM services WHERE name = 'RSPNDR'), 'DISABLED') = 'DISABLED';`,
	},
	{
		Name:        "CIS 4.11.36.4.2.1 (L2) Ensure 'Allow users to connect remotely by using Remote Desktop Services' is set to 'Disabled' (Automated)",
		Description: "This policy setting allows you to configure remote access to computers by using Remote Desktop Services. The recommended state for this setting is: Disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.11.36.4.2.1",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Terminal Server\fDenyTSConnections' AND data = '1';`,
	},
	{
		Name:        "CIS 4.11.36.4.3.3 (L2) Ensure 'Do not allow LPT port redirection' is set to 'Enabled' (Automated)",
		Description: "This policy setting specifies whether to prevent the redirection of data to client LPT ports during a Remote Desktop Services session. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.11.36.4.3.3",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services\fDisableLPT' AND data = '1';`,
	},
	{
		Name:        "CIS 4.11.54.1 (L2) Ensure 'Turn on PowerShell Script Block Logging' is set to 'Enabled' (Automated)",
		Description: "This policy setting enables logging of all PowerShell script input to the Applications and Services Logs event log channel. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.11.54.1",
		SQL:         `SELECT 1 FROM mdm_bridge WHERE mdm_command_input = '<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Result/WindowsPowerShell/TurnOnPowerShellScriptBlockLogging</LocURI></Target></Item></Get></SyncBody>' AND mdm_command_output LIKE '%<Enabled/>%';`,
	},
	{
		Name:        "CIS 4.11.55.2.2 (L2) Ensure 'Allow remote server management through WinRM' is set to 'Disabled' (Automated)",
		Description: "This policy setting allows you to manage whether the WinRM service automatically listens on the network for requests on the HTTP transport. The recommended state for this setting is: Disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.11.55.2.2",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\WinRM\Service\AllowAutoConfig' AND data = '0';`,
	},
	{
		Name:        "CIS 4.10.20.1.13 (L2) Ensure 'Turn off Windows Error Reporting' is set to 'Enabled' (Automated)",
		Description: "This policy setting controls whether or not errors are reported to Microsoft. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS4.10.20.1.13",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\Windows Error Reporting\Disabled' AND data = '1';`,
	},
	{
		Name:        "CIS 22.24 (L2) Ensure 'Enable Convert Warn To Block' is set to 'Warn verdicts are converted to block' (Automated)",
		Description: "This policy setting controls whether Microsoft Defender Antivirus network protection will display a warning or block network traffic. The recommended state for this setting is: Warn verdicts are converted to block.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS22.24",
		SQL:         `SELECT 1 FROM mdm_bridge WHERE mdm_command_input = '<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Defender/Configuration/EnableConvertWarnToBlock</LocURI></Target></Item></Get></SyncBody>' AND mdm_command_output = '1';`,
	},
	{
		Name:        "CIS 22.25 (L2) Ensure 'Enable File Hash Computation' is set to 'Enable' (Automated)",
		Description: "This setting determines whether hash values are computed for files scanned by Microsoft Defender. The recommended state for this setting is: Enable.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "L2",
		CISRef:      "CIS22.25",
		SQL:         `SELECT 1 FROM mdm_bridge WHERE mdm_command_input = '<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Defender/Configuration/EnableFileHashComputation</LocURI></Target></Item></Get></SyncBody>' AND mdm_command_output = '1';`,
	},

	// =========================================================================
	// WINDOWS — CIS-8.1 Win11 Intune (BL - BitLocker)
	// Source: https://github.com/karmine05/fleet_policies/blob/main/CIS-8.1/win11/intune/bl_win11_intune.yaml
	// =========================================================================
	{
		Name:        "CIS 8.1 (BL) Ensure 'Require Device Encryption' is set to 'Enabled' (Automated)",
		Description: "This setting allows the Admin to require encryption to be turned on using BitLocker/Device Encryption. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "BL",
		CISRef:      "CIS8.1",
		SQL:         `SELECT 1 FROM bitlocker_info WHERE protection_status = 1;`,
	},
	{
		Name:        "CIS 8.2 (BL) Ensure 'Allow Warning For Other Disk Encryption' is set to 'Disabled' (Automated)",
		Description: "This setting allows Admin to disable all UI and turn on encryption on the user machines silently. The recommended state for this setting is: Disabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "BL",
		CISRef:      "CIS8.2",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\PolicyManager\current\device\BitLocker\AllowWarningForOtherDiskEncryption' AND data = '0';`,
	},
	{
		Name:        "CIS 8.3 (BL) Ensure 'Allow Warning For Other Disk Encryption: Allow Standard User Encryption' is set to 'Enabled' (Automated)",
		Description: "This setting allows the Admin to require encryption to be turned on using BitLocker/Device Encryption for standard users. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "BL",
		CISRef:      "CIS8.3",
		SQL:         `SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\PolicyManager\Providers\%\default\Device\BitLocker\AllowStandardUserEncryption' AND data = '1';`,
	},
	{
		Name:        "CIS 28.1 (BL) Ensure 'Device Enumeration Policy' is set to 'Block all (most restrictive)' (Automated)",
		Description: "This policy provides additional security against external DMA-capable devices. The recommended state for this setting is: Block all (most restrictive).",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "BL",
		CISRef:      "CIS28.1",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\DeviceInstall\Settings\DeviceEnumerationPolicy' AND data = '0';`,
	},
	{
		Name:        "CIS 4.11.7.1.1 (BL) Ensure 'Choose how BitLocker-protected fixed drives can be recovered' is set to 'Enabled' (Automated)",
		Description: "This policy setting allows you to control how BitLocker-protected fixed data drives are recovered in the absence of the required credentials. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "BL",
		CISRef:      "CIS4.11.7.1.1",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\FVE\FDVRecovery' AND data = '1';`,
	},
	{
		Name:        "CIS 4.11.7.2.1 (BL) Ensure 'Choose how BitLocker-protected operating system drives can be recovered' is set to 'Enabled' (Automated)",
		Description: "This policy setting allows you to control how BitLocker-protected operating system drives are recovered. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "BL",
		CISRef:      "CIS4.11.7.2.1",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\FVE\OSRecovery' AND data = '1';`,
	},
	{
		Name:        "CIS 4.11.7.2.3 (BL) Ensure 'Choose how BitLocker-protected operating system drives can be recovered: Recovery Password' is set to 'Enabled: Require 48-digit recovery password' (Automated)",
		Description: "Configures whether users are allowed, required, or not allowed to generate a 48-digit recovery password or a 256-bit recovery key for OS drives. The recommended state: Enabled: Require 48-digit recovery password.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "BL",
		CISRef:      "CIS4.11.7.2.3",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\FVE\OSRecoveryPassword' AND data = '1';`,
	},
	{
		Name:        "CIS 4.11.7.3.1 (BL) Ensure 'Deny write access to removable drives not protected by BitLocker' is set to 'Enabled' (Automated)",
		Description: "This policy setting configures whether BitLocker protection is required for a computer to be able to write data to a removable data drive. The recommended state for this setting is: Enabled.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "BL",
		CISRef:      "CIS4.11.7.3.1",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\FVE\RDVDenyWriteAccess' AND data = '1';`,
	},
	{
		Name:        "CIS 4.11.7.4 (BL) Ensure 'Choose drive encryption method and cipher strength... fixed data drives' is set to 'XTS-AES 128-bit (default)' or 'XTS-AES 256-bit' (Automated)",
		Description: "This policy setting determines which encryption method should be used for fixed data drives. The recommended state for this setting is: XTS-AES 128-bit (default) or XTS-AES 256-bit.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "BL",
		CISRef:      "CIS4.11.7.4",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\FVE\EncryptionMethodWithXtsFdv' AND data = '7';`,
	},
	{
		Name:        "CIS 4.11.7.5 (BL) Ensure 'Choose drive encryption method and cipher strength... operating system drives' is set to 'XTS-AES 128-bit (default)' or 'XTS-AES 256-bit' (Automated)",
		Description: "This policy setting determines which encryption method should be used for operating system drives. The recommended state for this setting is: XTS-AES 128-bit (default) or XTS-AES 256-bit.",
		Platform:    "windows",
		Category:    "CIS",
		Level:       "BL",
		CISRef:      "CIS4.11.7.5",
		SQL:         `SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\FVE\EncryptionMethodWithXtsOs' AND data = '7';`,
	},
}

// GetVettedQueries returns vetted CIS/security queries filtered by platform.
// Platform: "darwin"/"macos", "windows", "linux", or "all".
func GetVettedQueries(platform string) []VettedQuery {
	p := strings.ToLower(strings.TrimSpace(platform))
	if p == "macos" || p == "mac" || p == "osx" {
		p = "darwin"
	}

	if p == "" || p == "all" {
		return vettedQueryLibrary
	}

	var result []VettedQuery
	for _, q := range vettedQueryLibrary {
		if q.Platform == p {
			result = append(result, q)
		}
	}
	return result
}
