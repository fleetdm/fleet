import { IPolicyNew } from "interfaces/policy";
import { SelectedPlatformString } from "interfaces/platform";

const DEFAULT_POLICY_PLATFORM: SelectedPlatformString = "";

export const DEFAULT_POLICY = {
  id: 1,
  name: "Is osquery running?",
  query: "SELECT 1 FROM osquery_info WHERE start_time > 1;",
  description: "Checks if the osquery process has started on the host.",
  author_id: 42,
  author_name: "John",
  author_email: "john@example.com",
  resolution: "Resolution steps",
  platform: DEFAULT_POLICY_PLATFORM,
  passing_host_count: 2000,
  failing_host_count: 300,
  created_at: "",
  updated_at: "",
  critical: false,
};

// We disable some linting and prettier for DEFAULT_POLICIES object because we
// need to keep some backslash(\) characters in some of the query string values.

/* eslint-disable no-useless-escape */
// prettier-ignore
export const DEFAULT_POLICIES: IPolicyNew[] = [
  {
    key: 1,
    query:
      "SELECT score FROM (SELECT case when COUNT(*) = 2 then 1 ELSE 0 END AS score FROM processes WHERE (name = 'clamd') OR (name = 'freshclam')) WHERE score == 1;",
    name: "Antivirus healthy (Linux)",
    description:
      "If ClamAV and Freshclam are not running, the workstation lacks active virus scanning, increasing malware infection risk.",
    resolution: "ClamAV and Freshclam will be checked and restarted if necessary, restoring virus protection.",
    critical: false,
    platform: "linux",
  },
  {
    key: 2,
    query:
      "SELECT score FROM (SELECT case when COUNT(*) = 2 then 1 ELSE 0 END AS score FROM plist WHERE (key = 'CFBundleShortVersionString' AND path = '/Library/Apple/System/Library/CoreServices/XProtect.bundle/Contents/Info.plist' AND value>=2162) OR (key = 'CFBundleShortVersionString' AND path = '/Library/Apple/System/Library/CoreServices/MRT.app/Contents/Info.plist' and value>=1.93)) WHERE score == 1;",
    name: "Antivirus healthy (macOS)",
    description:
      "If XProtect or MRT are not updated, the system risks exposure to malware not covered by older definitions.",
    resolution:
      "Update XProtect and MRT to the latest versions, bolstering your system's defense against new threats.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 3,
    query:
      "SELECT 1 from windows_security_center wsc CROSS JOIN windows_security_products wsp WHERE antivirus = 'Good' AND type = 'Antivirus' AND signatures_up_to_date=1;",
    name: "Antivirus healthy (Windows)",
    description:
      "Lack of active, updated antivirus exposes the workstation to malware and security threats.",
    resolution:
      "Ensure Windows Defender or your third-party antivirus is running, up to date, and visible in the Windows Security Center.",
    critical: false,
    platform: "windows",
  },
  {
    key: 4,
    query:
      "SELECT 1 FROM managed_policies WHERE domain = 'com.apple.loginwindow' AND name = 'com.apple.login.mcx.DisableAutoLoginClient' AND value = 1 LIMIT 1;",
    name: "Automatic login disabled (macOS)",
    description:
      "Auto-login being enabled increases risk of unauthorized access if the workstation is compromised.",
    resolution:
      "Auto-login will be disabled to secure the workstation against unauthorized use.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 5,
    query:
      "SELECT 1 FROM (SELECT encrypted, path FROM disk_encryption FULL OUTER JOIN mounts ON mounts.device_alias = disk_encryption.name) WHERE encrypted = 1 AND path = '/';",
    name: "Full disk encryption enabled (Linux)",
    description:
      "Unencrypted root filesystem means sensitive data might be easily accessible to unauthorized parties, increasing data breach risks.",
    resolution:
      "Ensure the image deployed to your Linux workstation includes full disk encryption.",
    critical: false,
    platform: "linux",
  },
  {
    key: 6,
    query:
      "SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT '' AND filevault_status = 'on' LIMIT 1;",
    name: "Full disk encryption enabled (macOS)",
    description:
      "If FileVault is off, the user's data is vulnerable to unauthorized access and potential data breaches.",
    resolution:
      "FileVault will be turned on to enable full disk encryption.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 7,
    query:
      "SELECT 1 FROM bitlocker_info WHERE drive_letter='C:' AND protection_status=1;",
    name: "Full disk encryption enabled (Windows)",
    description:
      "If BitLocker is disabled, the workstation's data is at risk of unauthorized access and theft.",
    resolution:
      "Full disk encryption will be enabled to secure data.",
    critical: false,
    platform: "windows",
  },
  {
    key: 8,
    query: "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
    name: "Gatekeeper enabled (macOS)",
    description:
      "Disabled Gatekeeper increases risk of installing potentially malicious apps.",
    resolution:
      "Gatekeeper will be enabled to ensure only trusted software is run on the device.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 9,
    query: "SELECT 1 FROM mdm WHERE enrolled='true';",
    name: "MDM enrolled (macOS)",
    description:
      "Workstations not enrolled to MDM miss critical security updates and remote management capabilities.",
    resolution: "Enroll device to MDM.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 10,
    query:
      "SELECT 1 FROM managed_policies WHERE domain = 'com.apple.Terminal' AND name = 'SecureKeyboardEntry' AND value = 1 LIMIT 1;",
    name: "Secure keyboard entry for Terminal application enabled (macOS)",
    description:
      "If secure keyboard entry is disabled, it increases vulnerability to keyloggers and other snooping software.",
    resolution:
      "Secure keyboard entry will be enabled to enhance protection against keystroke logging.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 11,
    query:
      "SELECT 1 FROM sip_config WHERE config_flag = 'sip' AND enabled = 1;",
    name: "System Integrity Protection enabled (macOS)",
    description:
      "Disabled System Integrity Protection increases risk of unauthorized system modifications and malware.",
    resolution:
      "System Integrity Protection will be enabled by running the following command: /usr/sbin/spctl --master-enable.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 12,
    query: "SELECT 1 FROM alf WHERE global_state >= 1;",
    name: "Firewall enabled (macOS)",
    description: "If the firewall is disabled, the workstation is vulnerable to unauthorized network access and attacks.",
    resolution:
      "The firewall will be enabled to protect against external threats.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 13,
    query:
      "SELECT 1 FROM managed_policies WHERE name='askForPassword' AND value='1';",
    name: "Screen lock enabled (macOS)",
    description:
      "Disabling password prompts increases the risk of unauthorized system access.",
    resolution:
      "Configuration changes will enforce immediate password prompts to mitigate unauthorized access risks.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 14,
    query:
      "SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\\Software\\Microsoft\\Windows\\CurrentVersion\\Policies\\System\\InactivityTimeoutSecs' AND CAST(data as INTEGER) <= 1800;",
    name: "Screen lock enabled (Windows)",
    description:
      "Devices with inactive timeout settings over 30 minutes risk prolonged unauthorized access if left unattended, exposing sensitive data.",
    resolution:
      "Enable the Interactive Logon: Machine inactivity limit setting with a value of 1800 seconds or lower.",
    critical: false,
    platform: "windows",
  },
  {
    key: 15,
    query:
      "SELECT 1 FROM (SELECT cast(lengthtxt as integer(2)) minlength FROM (SELECT SUBSTRING(length, 1, 2) AS lengthtxt FROM (SELECT policy_description, policy_identifier, split(policy_content, '{', 1) AS length FROM password_policy WHERE policy_identifier LIKE '%minLength')) WHERE minlength >= 10);",
    name: "Password requires 10 or more characters (macOS)",
    description:
      "Password policies requiring less than 10 characters increase vulnerability to brute-force attacks",
    resolution:
      "Password requirements will be strengthened to a minimum of 10 characters.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 16,
    query: "SELECT 1 FROM os_version WHERE version >= '14.6.1' OR version >= '15.0';",
    name: "Operating system up to date (macOS)",
    description: "Using an outdated macOS version risks exposure to security vulnerabilities and potential system instability.",
    resolution:
      "We will update your macOS to the latest version to enhance security and stability.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 17,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.SoftwareUpdate' AND name='AutomaticCheckEnabled' AND value=1 LIMIT 1;",
    name: "Automatic updates enabled (macOS)",
    description:
      "Checks that a mobile device management (MDM) solution configures the Mac to automatically check for updates.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that enables automatic updates.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 18,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.SoftwareUpdate' AND name='AutomaticDownload' AND value=1 LIMIT 1;",
    name: "Automatic update downloads enabled (macOS)",
    description:
      "Checks that a mobile device management (MDM) solution configures the Mac to automatically download updates.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that enables automatic update downloads.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 19,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.SoftwareUpdate' AND name='AutomaticallyInstallAppUpdates' AND value=1 LIMIT 1;",
    name: "Installation of application updates is enabled (macOS)",
    description:
      "When the Mac is not configureed to automatically install updates to Apple applications, this risks security vulnerabilities and potential exploitation.",
    resolution:
      "The automatic software update feature will be enabled to ensure that the workstation receives timely updates.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 20,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.SoftwareUpdate' AND name='CriticalUpdateInstall' AND value=1 LIMIT 1;",
    name: "Automatic security and data file updates is enabled (macOS)",
    description:
      "If the Mac is not automatically downloading updates to built-in macOS security tools, critical updates may not be installed, leaving the device vulnerable to potential exploitation.",
    resolution:
      "Enable automatic security and data update installation.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 21,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.SoftwareUpdate' AND name='AutomaticallyInstallMacOSUpdates' AND value=1 LIMIT 1;",
    name:
      "Automatic installation of operating system updates is enabled (macOS)",
    description:
      "If automatic macOS updates are not enabled, critical updates may not be installed, leaving the device vulnerable to potential exploitation.",
    resolution:
      "Enable automatic installation of operating system updates.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 22,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.applicationaccess' AND name='forceAutomaticDateAndTime' AND value=1 LIMIT 1;",
    name: "Time and date are configured to be updated automatically (macOS)",
    description:
      "If the automatic setting of date and time is disabled, there could be synchronization issues with other systems, services, or applications.",
    resolution:
      "Enable automatic time and date configuration.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 23,
    query:
      "SELECT 1 WHERE EXISTS (SELECT CAST(value as integer(4)) valueint from managed_policies WHERE domain = 'com.apple.screensaver' AND name = 'askForPasswordDelay' AND valueint <= 60 LIMIT 1) AND EXISTS (SELECT CAST(value as integer(4)) valueint from managed_policies WHERE domain = 'com.apple.screensaver' AND name = 'idleTime' AND valueint <= 1140 LIMIT 1) AND EXISTS (SELECT 1 from managed_policies WHERE domain='com.apple.screensaver' AND name='askForPassword' AND value=1 LIMIT 1);",
    name: "Lock screen after inactivity of 20 minutes or less (macOS)",
    description:
      "Inadequate screen saver security settings could potentially allow unauthorized access to the workstation if left unattended for extended periods.",
    resolution:
      "Ensure screen saver is enabled after inactivity of 20 minutes or less.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 24,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.MCX' AND name='forceInternetSharingOff' AND value='1' LIMIT 1;",
    name: "Internet sharing blocked (macOS)",
    description:
      "Unauthorized Internet sharing could potentially expose sensitive network resources to external threats.",
    resolution:
      "The Internet sharing setting will be disabled",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 25,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.applicationaccess' AND name='allowContentCaching' AND value='0' LIMIT 1;",
    name: "Content caching is disabled (macOS)",
    description:
      "Enabling content caching could lead to unauthorized caching of sensitive data, potentially exposing it to unauthorized access.",
    resolution:
      "Content caching will be disabled.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 26,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.AdLib' AND name='forceLimitAdTracking' AND value='1' LIMIT 1;",
    name: "Ad tracking is limited (macOS)",
    description:
      "Failure to limit ad tracking could result in excessive tracking of user behavior and preferences by advertisers, compromising privacy.",
    resolution:
      "Advertisement tracking will be disabled.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 27,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.icloud.managed' AND name='DisableCloudSync' AND value='1' LIMIT 1;",
    name: "iCloud Desktop and Document sync is disabled (macOS)",
    description:
      "Checks that a mobile device management (MDM) solution configures the Mac to prevent iCloud Desktop and Documents sync.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile to prevent iCloud Desktop and Documents sync.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 28,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.security.firewall' AND name='EnableLogging' AND value='1' LIMIT 1;",
    name: "Firewall logging is enabled (macOS)",
    description:
      "Without firewall logging enabled, it becomes difficult to monitor and track network traffic, increasing the risk of undetected malicious activities or unauthorized access.",
    resolution:
      "Firewall logging will be enabled on the workstation.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 29,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.loginwindow' AND name='DisableGuestAccount' AND value='1' LIMIT 1;",
    name: "Guest account disabled (macOS)",
    description:
      "Use of the guest account could allow unauthorized users to access the system, potentially leading to unauthorized access to sensitive data and security breaches.",
    resolution:
      "The guest account will be disabled.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 30,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.AppleFileServer' AND name='guestAccess' AND value='0' LIMIT 1;",
    name: "Guest access to shared folders is disabled (macOS)",
    description:
      "Guest access to shared folders could allow unauthorized users to access sensitive files and data, potentially leading to data breaches or unauthorized modifications.",
    resolution:
      "Guest access to shared folders will be disabled.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 31,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\SharedAccess\\Parameters\\FirewallPolicy\\DomainProfile\\EnableFirewall' AND CAST(data as integer) = 1;",
    name: "Windows Firewall, domain profile enabled (Windows)",
    description:
      "If the Windows Firewall is not enabled for the domain profile, the workstation may be more vulnerable to unauthorized network access and potential security breaches.",
    resolution:
      "The Windows Firewall will be enabled for the domain profile.",
    critical: false,
    platform: "windows",
  },
  {
    key: 32,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\SharedAccess\\Parameters\\FirewallPolicy\\StandardProfile\\EnableFirewall' AND CAST(data as integer) = 1;",
    name: "Windows Firewall, private profile enabled (Windows)",
    description:
      "If the Windows Firewall is not enabled for the private profile, the workstation may be more susceptible to unauthorized access and potential security breaches, particularly when connected to private networks.",
    resolution:
      "The Windows Firewall will be enabled for the private profile",
    critical: false,
    platform: "windows",
  },
  {
    key: 33,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\SharedAccess\\Parameters\\FirewallPolicy\\PublicProfile\\EnableFirewall' AND CAST(data as integer) = 1;",
    name: "Windows Firewall, public profile enabled (Windows)",
    description:
      "If the Windows Firewall is not enabled for the public profile, the workstation may be more vulnerable to unauthorized access and potential security threats, especially when connected to public networks.",
    resolution:
      "The Windows Firewall will be enabled for the public profile.",
    critical: false,
    platform: "windows",
  },
  {
    key: 34,
    query:
      "SELECT 1 FROM windows_optional_features WHERE name = 'SMB1Protocol-Client' AND state != 1;",
    name: "SMBv1 client driver disabled (Windows)",
    description: "Leaving the SMBv1 client enabled increases vulnerability to security threats and potential exploitation by malicious actors.",
    resolution:
      "The SMBv1 client will be disabled.",
    critical: false,
    platform: "windows",
  },
  {
    key: 35,
    query:
      "SELECT 1 FROM windows_optional_features WHERE name = 'SMB1Protocol-Server' AND state != 1",
    name: "SMBv1 server disabled (Windows)",
    description: "Leaving the SMBv1 server enabled exposes the workstation to potential security vulnerabilities and exploitation by malicious actors.",
    resolution:
      "The SMBv1 server will be disabled.",
    critical: false,
    platform: "windows",
  },
  {
    key: 36,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\\Software\\Policies\\Microsoft\\Windows NT\\DNSClient\\EnableMulticast' AND CAST(data as integer) = 0;",
    name: "LLMNR disabled (Windows)",
    description:
      "If the workstation does not have LLMNR disabled, it could be vulnerable to DNS spoofing attacks, potentially leading to unauthorized access or data interception.",
    resolution:
      "LLMNR will be disabled on your system.",
    critical: false,
    platform: "windows",
  },
  {
    key: 37,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\\Software\\Policies\\Microsoft\\Windows\\Windows\\Update\\AU\\NoAutoUpdate' AND CAST(data as integer) = 0;",
    name: "Automatic updates enabled (Windows)",
    description:
      "Enabling automatic updates ensures the computer downloads and installs security and other important updates automatically.",
    resolution:
      "Automatic updates will be enabled.",
    critical: false,
    platform: "windows",
  },
  {
    key: 38,
    query:
      "SELECT EXISTS(SELECT 1 FROM file WHERE filename like '%Emergency Kit%.pdf' AND (path LIKE '/Users/%%/Downloads/%%' OR path LIKE '/Users/%%/Desktop/%%')) as does_1p_ek_exist;",
    name:
      "No 1Password emergency kit stored on desktop or in downloads (macOS)",
    description:
      "Storing the 1Password emergency kit on the desktop or in the downloads folder increases the risk of unauthorized access to sensitive credentials if the workstation is compromised or accessed by unauthorized users.",
    resolution:
      "1Password emergency kits must be printed and stored in a physically secure location.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 39,
    query:
      "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM users CROSS JOIN user_ssh_keys USING (uid) WHERE encrypted='0');",
    name: "No unencrypted SSH keys present",
    description: "Having unencrypted SSH keys poses the risk of unauthorized access to sensitive systems and data if the workstation is compromised.",
    resolution:
      "Any unencrypted SSH keys will be encrypted or removed from the workstation.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 40,
    query:
      "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM keychain_items WHERE label LIKE '%ABCDEFG%' LIMIT 1);",
    name: "No Apple signing or notarization credentials secrets stored (macOS)",
    description:
      "Storing Apple signing or notarization credentials poses the risk of unauthorized access to sensitive development assets and potential compromise of software integrity.",
    resolution:
      "Apple signing or notarization credentials secrets will be removed from the workstation.",
    critical: false,
    platform: "darwin",
  },
];
