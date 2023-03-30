import { IPolicyNew } from "interfaces/policy";
import { IPlatformString } from "interfaces/platform";

const DEFAULT_POLICY_PLATFORM: IPlatformString = "";

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
      "Checks that both ClamAV's daemon and its updater service (freshclam) are running.",
    resolution: "Ensure ClamAV and Freshclam are installed and running.",
    critical: false,
    platform: "linux",
  },
  {
    key: 2,
    query:
      "SELECT score FROM (SELECT case when COUNT(*) = 2 then 1 ELSE 0 END AS score FROM plist WHERE (key = 'CFBundleShortVersionString' AND path = '/Library/Apple/System/Library/CoreServices/XProtect.bundle/Contents/Info.plist' AND value>=2162) OR (key = 'CFBundleShortVersionString' AND path = '/Library/Apple/System/Library/CoreServices/MRT.app/Contents/Info.plist' and value>=1.93)) WHERE score == 1;",
    name: "Antivirus healthy (macOS)",
    description:
      "Checks the version of Malware Removal Tool (MRT) and the built-in macOS AV (Xprotect). Replace version numbers with the latest version regularly.",
    resolution:
      "To enable automatic security definition updates, on the failing device, select System Preferences > Software Update > Advanced > Turn on Install system data files and security updates.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 3,
    query:
      "SELECT 1 from windows_security_center wsc CROSS JOIN windows_security_products wsp WHERE antivirus = 'Good' AND type = 'Antivirus' AND signatures_up_to_date=1;",
    name: "Antivirus healthy (Windows)",
    description:
      "Checks the status of antivirus and signature updates from the Windows Security Center.",
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
      "Checks that a mobile device management (MDM) solution configures the Mac to prevent log in without a password.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that disables automatic login.",
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
      "Checks if the device mounted at / is encrypted. There are many ways to encrypt Linux systems. You may need to adapt this query, or submit an issue in the Fleet repo.",
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
      "Checks to make sure that full disk encryption (FileVault) is enabled on macOS devices.",
    resolution:
      "To enable full disk encryption, on the failing device, select System Preferences > Security & Privacy > FileVault > Turn On FileVault.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 7,
    query:
      "SELECT 1 FROM bitlocker_info WHERE drive_letter='C:' AND protection_status=1;",
    name: "Full disk encryption enabled (Windows)",
    description:
      "Checks to make sure that full disk encryption is enabled on Windows devices.",
    resolution:
      "To get additional information, run the following osquery query on the failing device: SELECT * FROM bitlocker_info. In the query results, if protection_status is 2, then the status cannot be determined. If it is 0, it is considered unprotected. Use the additional results (percent_encrypted, conversion_status, etc.) to help narrow down the specific reason why Windows considers the volume unprotected.",
    critical: false,
    platform: "windows",
  },
  {
    key: 8,
    query: "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
    name: "Gatekeeper enabled (macOS)",
    description:
      "Checks to make sure that the Gatekeeper feature is enabled on macOS devices. Gatekeeper tries to ensure only trusted software is run on a mac machine.",
    resolution:
      "To enable Gatekeeper, on the failing device, run the following command in the Terminal app: /usr/sbin/spctl --master-enable.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 9,
    query: "SELECT 1 FROM mdm WHERE enrolled='true';",
    name: "MDM enrolled (macOS)",
    description:
      "Required: osquery deployed with Orbit, or manual installation of macadmins/osquery-extension. Checks that a Mac is enrolled to MDM. Add a AND on identity_certificate_uuid to check for a specific MDM.",
    resolution: "Enroll device to MDM",
    critical: false,
    platform: "darwin",
  },
  {
    key: 10,
    query:
      "SELECT 1 FROM managed_policies WHERE domain = 'com.apple.Terminal' AND name = 'SecureKeyboardEntry' AND value = 1 LIMIT 1;",
    name: "Secure keyboard entry for Terminal application enabled (macOS)",
    description:
      "Checks that a mobile device management (MDM) solution configures the Mac to enabled secure keyboard entry for the Terminal application.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that enables secure keyboard entry for the Terminal application.",
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
      "Checks to make sure that the System Integrity Protection feature is enabled.",
    resolution:
      "To enable System Integrity Protection, on the failing device, run the following command in the Terminal app: /usr/sbin/spctl --master-enable.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 12,
    query: "SELECT 1 FROM alf WHERE global_state >= 1;",
    name: "Firewall enabled (macOS)",
    description: "Checks if the firewall is enabled.",
    resolution:
      "In System Preferences, open Security & Privacy, navigate to the Firewall tab and click Turn On Firewall.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 13,
    query:
      "SELECT 1 FROM managed_policies WHERE name='askForPassword' AND value='1';",
    name: "Screen lock enabled (macOS)",
    description:
      "Checks that a mobile device management (MDM) solution configures the Mac to enable screen lock.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that enables screen lock.",
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
      "Checks if the screen lock is enabled and configured to lock the system within 30 minutes or less.",
    resolution:
      "Contact your IT administrator to enable the Interactive Logon: Machine inactivity limit setting with a value of 1800 seconds or lower.",
    critical: false,
    platform: "windows",
  },
  {
    key: 15,
    query:
      "SELECT 1 FROM (SELECT cast(lengthtxt as integer(2)) minlength FROM (SELECT SUBSTRING(length, 1, 2) AS lengthtxt FROM (SELECT policy_description, policy_identifier, split(policy_content, '{', 1) AS length FROM password_policy WHERE policy_identifier LIKE '%minLength')) WHERE minlength >= 10);",
    name: "Password requires 10 or more characters (macOS)",
    description:
      "Checks that the password policy requires at least 10 characters. Requires osquery 5.4.0 or newer.",
    resolution:
      "Contact your IT administrator to confirm that your Mac is receiving configuration profiles for password length.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 16,
    query: "SELECT 1 FROM os_version WHERE version >= '12.5.1';",
    name: "Operating system up to date (macOS)",
    description: "Checks that the operating system is up to date.",
    resolution:
      "From the Apple menu () in the corner of your screen choose System Preferences. Then select Software Update and select Upgrade Now. You might be asked to restart or enter your password.",
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
      "Checks that a mobile device management (MDM) solution configures the Mac to automatically install updates to Apple applications.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that enables installation of application updates.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 20,
    query:
      "SELECT 1 FROM managed_policies WHERE domain='com.apple.SoftwareUpdate' AND name='CriticalUpdateInstall' AND value=1 LIMIT 1;",
    name: "Automatic security and data file updates is enabled (macOS)",
    description:
      "Checks that a mobile device management (MDM) solution configures the Mac to automatically download updates to built-in macOS security tools such as malware removal tools.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that enables automatic security and data update installation.",
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
      "Checks that a mobile device management (MDM) solution configures the Mac to automatically install operating system updates.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that enables automatic installation of operating system updates.",
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
      "Checks that a mobile device management (MDM) solution configures the Mac to automatically update the time and date.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that enables automatic time and date configuration.",
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
      "Checks that a mobile device management (MDM) solution configures the Mac to lock the screen after 20 minutes or less.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that enables the screen saver after inactivity of 20 minutes or less.",
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
      "Checks that a mobile device management (MDM) solution configures the Mac to prevent Internet sharing.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that prevents Internet sharing.",
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
      "Checks that a mobile device management (MDM) solution configures the Mac to disable content caching.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that disables content caching.",
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
      "Checks that a mobile device management (MDM) solution configures the Mac to limit advertisement tracking.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that disables advertisement tracking.",
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
      "Checks that a mobile device management (MDM) solution configures the Mac to log firewall activity.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that enables firewall logging.",
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
      "Checks that a mobile device management (MDM) solution configures the Mac to prevent the use of a guest account.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that disables the guest account.",
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
      "Checks that a mobile device management (MDM) solution configures the Mac to prevent guest access to shared folders.",
    resolution:
      "Contact your IT administrator to ensure your Mac is receiving a profile that prevents guest access to shared folders.",
    critical: false,
    platform: "darwin",
    mdm_required: true,
  },
  {
    key: 31,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINESoftwarePoliciesMicrosoftWindowsFirewallDomainProfileEnableFirewall' AND CAST(data as integer) = 1;",
    name: "Windows Firewall, Domain Profile enabled (Windows)",
    description:
      "Checks if a Group Policy configures the computer to enable the domain profile for Windows Firewall. The domain profile applies to networks where the host system can authenticate to a domain controller. Some auditors require that this setting is configured by a Group Policy.",
    resolution:
      "Contact your IT administrator to ensure your computer is receiving a Group Policy that enables the domain profile for Windows Firewall.",
    critical: false,
    platform: "windows",
  },
  {
    key: 32,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINESoftwarePoliciesMicrosoftWindowsFirewallPrivateProfileEnableFirewall' AND CAST(data as integer) = 1;",
    name: "Windows Firewall, Private Profile enabled (Windows)",
    description:
      "Checks if a Group Policy configures the computer to enable the private profile for Windows Firewall. The private profile applies to networks where the host system is connected to a private or home network. Some auditors require that this setting is configured by a Group Policy.",
    resolution:
      "Contact your IT administrator to ensure your computer is receiving a Group Policy that enables the private profile for Windows Firewall.",
    critical: false,
    platform: "windows",
  },
  {
    key: 33,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINESoftwarePoliciesMicrosoftWindowsFirewallPublicProfileEnableFirewall' AND CAST(data as integer) = 1;",
    name: "Windows Firewall, Public Profile enabled (Windows)",
    description:
      "Checks if a Group Policy configures the computer to enable the public profile for Windows Firewall. The public profile applies to networks where the host system is connected to public networks such as Wi-Fi hotspots at coffee shops and airports. Some auditors require that this setting is configured by a Group Policy.",
    resolution:
      "Contact your IT administrator to ensure your computer is receiving a Group Policy that enables the public profile for Windows Firewall.",
    critical: false,
    platform: "windows",
  },
  {
    key: 34,
    query:
      "SELECT 1 FROM windows_optional_features WHERE name = 'SMB1Protocol-Client' AND state != 1;",
    name: "SMBv1 client driver disabled (Windows)",
    description: "Checks that the SMBv1 client is disabled.",
    resolution:
      "Contact your IT administrator to discuss disabling SMBv1 on your system.",
    critical: false,
    platform: "windows",
  },
  {
    key: 35,
    query:
      "SELECT 1 FROM windows_optional_features WHERE name = 'SMB1Protocol-Server' AND state != 1",
    name: "SMBv1 server disabled (Windows)",
    description: "Checks that the SMBv1 server is disabled.",
    resolution:
      "Contact your IT administrator to discuss disabling SMBv1 on your system.",
    critical: false,
    platform: "windows",
  },
  {
    key: 36,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINESOFTWAREPoliciesMicrosoftWindows NTDNSClientEnableMulticast' AND CAST(data as integer) = 0;",
    name: "LLMNR disabled (Windows)",
    description:
      "Checks if a Group Policy configures the computer to disable LLMNR. Some auditors requires that this setting is configured by a Group Policy.",
    resolution:
      "Contact your IT administrator to ensure your computer is receiving a Group Policy that disables LLMNR on your system.",
    critical: false,
    platform: "windows",
  },
  {
    key: 37,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINESoftwarePoliciesMicrosoftWindowsWindowsUpdateAUNoAutoUpdate' AND CAST(data as integer) = 0;",
    name: "Automatic updates enabled (Windows)",
    description:
      "Checks if a Group Policy configures the computer to enable Automatic Updates. When enabled, the computer downloads and installs security and other important updates automatically. Some auditors requires that this setting is configured by a Group Policy.",
    resolution:
      "Contact your IT administrator to ensure your computer is receiving a Group policy that enables Automatic Updates.",
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
      "Looks for PDF files with file names typically used by 1Password for emergency recovery kits.",
    resolution:
      "Delete 1Password emergency kits from your computer, and empty the trash. 1Password emergency kits should only be printed and stored in a physically secure location.",
    critical: false,
    platform: "darwin",
  },
  {
    key: 39,
    query:
      "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM users CROSS JOIN user_ssh_keys USING (uid) WHERE encrypted='0');",
    name: "No unencrypted SSH keys present",
    description: "Checks if unencrypted SSH keys are present on the system.",
    resolution:
      "Remove SSH keys that are not necessary, and encrypt those that are. On Mac and Linux, use this command to encrypt your existing SSH keys: ssh-keygen -o -p -f path/to/keyfile",
    critical: false,
    platform: "darwin",
  },
  {
    key: 40,
    query:
      "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM keychain_items WHERE label LIKE '%ABCDEFG%' LIMIT 1);",
    name: "No Apple signing or notarization credentials secrets stored (macOS)",
    description:
      "Looks for certificate material linked to a company's Apple Developer account, which should only be present on build servers and not workstations. Replace *ABCDEFG* with your company's identifier.",
    resolution:
      "Ensure your official Apple builds, signing and notarization happen on a centralized system, and remove these certificates from workstations.",
    critical: false,
    platform: "darwin",
  },
];
