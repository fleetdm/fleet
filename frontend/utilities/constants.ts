import URL_PREFIX from "router/url_prefix";
import { IOsqueryPlatform, IPlatformString } from "interfaces/platform";
import { IPolicyNew } from "interfaces/policy";

const { origin } = global.window.location;
export const BASE_URL = `${origin}${URL_PREFIX}/api`;

export enum PolicyResponse {
  PASSING = "passing",
  FAILING = "failing",
}

export const DEFAULT_GRAVATAR_LINK =
  "https://fleetdm.com/images/permanent/icon-avatar-default-128x128-2x.png";

export const DEFAULT_POLICIES = [
  {
    key: 1,
    query:
      "SELECT score FROM (SELECT case when COUNT(*) = 2 then 1 ELSE 0 END AS score FROM processes WHERE (name = 'clamd') OR (name = 'freshclam')) WHERE score == 1;",
    name: "Antivirus healthy (Linux)",
    description:
      "Checks that both ClamAV's daemon and its updater service (freshclam) are running.",
    resolution: "Ensure ClamAV and Freshclam are installed and running.",
    platform: "linux",
  },
  {
    key: 2,
    query:
      "SELECT 1 FROM managed_policies WHERE domain = 'com.apple.Terminal' AND name = 'SecureKeyboardEntry' AND value = 1 LIMIT 1;",
    name: "Antivirus healthy (macOS)",
    description:
      "Checks the version of Malware Removal Tool (MRT) and the built-in macOS AV (Xprotect). Replace version numbers with the latest version regularly.",
    resolution:
      "To enable automatic security definition updates, on the failing device, select System Preferences > Software Update > Advanced > Turn on Install system data files and security updates.",
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
    platform: "windows",
  },
  {
    key: 4,
    query:
      "SELECT 1 FROM managed_policies WHERE domain = 'com.apple.loginwindow' AND name = 'com.apple.login.mcx.DisableAutoLoginClient' AND value = 1 LIMIT 1;",
    name: "Automatic login disabled (macOS)",
    description:
      "Required: You’re already enforcing a policy via Mobile Device Management (MDM). Checks to make sure that the device user cannot log in to the device without a password.",
    resolution:
      "The following example profile includes a setting to disable automatic login: https://github.com/gregneagle/profiles/blob/fecc73d66fa17b6fa78b782904cb47cdc1913aeb/loginwindow.mobileconfig#L64-L65.",
    platform: "darwin",
  },
  {
    key: 5,
    query:
      "SELECT 1 FROM disk_encryption WHERE encrypted=1 AND name LIKE '/dev/dm-1';",
    name: "Full disk encryption enabled (Linux)",
    description:
      "Checks if the dm-1 device is encrypted. There are many ways to encrypt Linux systems. This is the default on distributions such as Ubuntu. You may need to adapt this query, or submit an issue in the Fleet repo.",
    resolution:
      "Ensure the image deployed to your Linux workstation includes full disk encryption.",
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
    platform: "darwin",
  },
  {
    key: 7,
    query: "SELECT 1 FROM bitlocker_info WHERE protection_status = 1;",
    name: "Full disk encryption enabled (Windows)",
    description:
      "Checks to make sure that full disk encryption is enabled on Windows devices.",
    resolution:
      "To get additional information, run the following osquery query on the failing device: SELECT * FROM bitlocker_info. In the query results, if protection_status is 2, then the status cannot be determined. If it is 0, it is considered unprotected. Use the additional results (percent_encrypted, conversion_status, etc.) to help narrow down the specific reason why Windows considers the volume unprotected.",
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
    platform: "darwin",
  },
  {
    key: 9,
    query:
      "SELECT 1 FROM managed_policies WHERE domain = 'com.apple.loginwindow' AND name = 'DisableGuestAccount' AND value = 1 LIMIT 1;",
    name: "Guest users disabled (macOS)",
    description:
      "Required: You’re already enforcing a policy via Mobile Device Management (MDM). Checks to make sure that guest accounts cannot be used to log in to the device without a password.",
    resolution:
      "The following example profile includes a setting to disable guest users: https://github.com/gregneagle/profiles/blob/fecc73d66fa17b6fa78b782904cb47cdc1913aeb/loginwindow.mobileconfig#L68-L71.",
    platform: "darwin",
  },
  {
    key: 10,
    query: "SELECT 1 FROM mdm WHERE enrolled='true';",
    name: "MDM enrolled (macOS)",
    description:
      "Required: osquery deployed with Orbit, or manual installation of macadmins/osquery-extension. Checks that a Mac is enrolled to MDM. Add a AND on identity_certificate_uuid to check for a specific MDM.",
    resolution: "Enroll device to MDM",
    platform: "darwin",
  },
  {
    key: 11,
    query:
      "SELECT 1 FROM managed_policies WHERE domain = 'com.apple.Terminal' AND name = 'SecureKeyboardEntry' AND value = 1 LIMIT 1;",
    name: "Secure keyboard entry for Terminal.app enabled (macOS)",
    description:
      "Required: You’re already enforcing a policy via Mobile Device Management (MDM). Checks to make sure that the Secure Keyboard Entry setting is enabled.",
    resolution: "",
    platform: "darwin",
  },
  {
    key: 12,
    query:
      "SELECT 1 FROM sip_config WHERE config_flag = 'sip' AND enabled = 1;",
    name: "System Integrity Protection enabled (macOS)",
    description:
      "Checks to make sure that the System Integrity Protection feature is enabled.",
    resolution:
      "To enable System Integrity Protection, on the failing device, run the following command in the Terminal app: /usr/sbin/spctl --master-enable.",
    platform: "darwin",
  },
  {
    key: 13,
    query: "SELECT 1 FROM alf WHERE global_state >= 1;",
    name: "Firewall enabled (macOS)",
    description: "Checks if the firewall is enabled.",
    resolution:
      "In System Preferences, open Security & Privacy, navigate to the Firewall tab and click Turn On Firewall.",
    platform: "darwin",
  },
  {
    key: 14,
    query:
      "SELECT 1 FROM managed_policies WHERE name='askForPassword' AND value='1';",
    name: "Screen lock enabled via MDM profile (macOS)",
    description: "Checks that a MDM profile configures the screen lock",
    resolution:
      "Contact your IT administrator to help you enroll your computer in your organization's MDM. If already enrolled, ask your IT administrator to enable the screen lock feature in the profile configuration.",
    platform: "darwin",
  },
  {
    key: 15,
    query:
      "SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\\Software\\Microsoft\\Windows\\CurrentVersion\\Policies\\System\\InactivityTimeoutSecs' AND CAST(data as INTEGER) <= 1800;",
    name: "Screen lock enabled (Windows)",
    description:
      "Checks if the screen lock is enabled and configured to lock the system within 30 minutes or less.",
    resolution:
      "Ask your IT administrator to enable the Interactive Logon: Machine inactivity limit setting with a value of 1800 seconds or lower.",
    platform: "windows",
  },
  {
    key: 16,
    query:
      "SELECT 1 FROM (SELECT cast(lengthtxt as integer(2)) minlength FROM (SELECT SUBSTRING(length, 1, 2) AS lengthtxt FROM (SELECT policy_description, policy_identifier, split(policy_content, '{', 1) AS length FROM password_policy WHERE policy_identifier LIKE '%minLength')) WHERE minlength >= 10);",
    name: "Password requires 10 or more characters (macOS)",
    description:
      "Checks that the password policy requires at least 10 characters. Requires osquery 5.4.0 or newer.",
    resolution:
      "Contact your IT administrator to confirm that your Mac is receiving configuration profiles for password length.",
    platform: "darwin",
  },
] as IPolicyNew[];

export const FREQUENCY_DROPDOWN_OPTIONS = [
  { value: 900, label: "Every 15 minutes" },
  { value: 3600, label: "Every hour" },
  { value: 21600, label: "Every 6 hours" },
  { value: 43200, label: "Every 12 hours" },
  { value: 86400, label: "Every day" },
  { value: 604800, label: "Every week" },
];

export const GITHUB_NEW_ISSUE_LINK =
  "https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md";

export const LOGGING_TYPE_OPTIONS = [
  { label: "Snapshot", value: "snapshot" },
  { label: "Differential", value: "differential" },
  {
    label: "Differential (ignore removals)",
    value: "differential_ignore_removals",
  },
];

export const MAX_OSQUERY_SCHEDULED_QUERY_INTERVAL = 604800;

export const MIN_OSQUERY_VERSION_OPTIONS = [
  { label: "All", value: "" },
  { label: "5.4.0 +", value: "5.4.0" },
  { label: "5.3.0 +", value: "5.3.0" },
  { label: "5.2.3 +", value: "5.2.4" },
  { label: "5.2.2 +", value: "5.2.2" },
  { label: "5.2.1 +", value: "5.2.1" },
  { label: "5.2.0 +", value: "5.2.0" },
  { label: "5.1.0 +", value: "5.1.0" },
  { label: "5.0.1 +", value: "5.0.1" },
  { label: "5.0.0 +", value: "5.0.0" },
  { label: "4.9.0 +", value: "4.9.0" },
  { label: "4.8.0 +", value: "4.8.0" },
  { label: "4.7.0 +", value: "4.7.0" },
  { label: "4.6.0 +", value: "4.6.0" },
  { label: "4.5.1 +", value: "4.5.1" },
  { label: "4.5.0 +", value: "4.5.0" },
  { label: "4.4.0 +", value: "4.4.0" },
  { label: "4.3.0 +", value: "4.3.0" },
  { label: "4.2.0 +", value: "4.2.0" },
  { label: "4.1.2 +", value: "4.1.2" },
  { label: "4.1.1 +", value: "4.1.1" },
  { label: "4.1.0 +", value: "4.1.0" },
  { label: "4.0.2 +", value: "4.0.2" },
  { label: "4.0.1 +", value: "4.0.1" },
  { label: "4.0.0 +", value: "4.0.0" },
  { label: "3.4.0 +", value: "3.4.0" },
  { label: "3.3.2 +", value: "3.3.2" },
  { label: "3.3.1 +", value: "3.3.1" },
  { label: "3.2.6 +", value: "3.2.6" },
  { label: "2.2.1 +", value: "2.2.1" },
  { label: "2.2.0 +", value: "2.2.0" },
  { label: "2.1.2 +", value: "2.1.2" },
  { label: "2.1.1 +", value: "2.1.1" },
  { label: "2.0.0 +", value: "2.0.0" },
  { label: "1.8.2 +", value: "1.8.2" },
  { label: "1.8.1 +", value: "1.8.1" },
];

export const QUERIES_PAGE_STEPS = {
  1: "EDITOR",
  2: "TARGETS",
  3: "RUN",
};

export const DEFAULT_QUERY = {
  description: "",
  name: "",
  query: "SELECT * FROM osquery_info;",
  id: 0,
  interval: 0,
  last_excuted: "",
  observer_can_run: false,
  author_name: "",
  updated_at: "",
  created_at: "",
  saved: false,
  author_id: 0,
  packs: [],
};

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
};

export const DEFAULT_CAMPAIGN = {
  created_at: "",
  errors: [],
  hosts: [],
  hosts_count: {
    total: 0,
    successful: 0,
    failed: 0,
  },
  id: 0,
  query_id: 0,
  query_results: [],
  status: "",
  totals: {
    count: 0,
    missing_in_action: 0,
    offline: 0,
    online: 0,
  },
  updated_at: "",
  user_id: 0,
};

export const DEFAULT_CAMPAIGN_STATE = {
  observerShowSql: false,
  queryIsRunning: false,
  queryPosition: {},
  queryResultsToggle: null,
  runQueryMilliseconds: 0,
  selectRelatedHostTarget: false,
  targetsCount: 0,
  targetsError: null,
  campaign: { ...DEFAULT_CAMPAIGN },
};

export const PLATFORM_DISPLAY_NAMES: Record<string, IOsqueryPlatform> = {
  darwin: "macOS",
  freebsd: "FreeBSD",
  linux: "Linux",
  windows: "Windows",
};

// as returned by the TARGETS API; based on display_text
export const PLATFORM_LABEL_DISPLAY_NAMES: Record<string, string> = {
  "All Hosts": "All hosts",
  "All Linux": "Linux",
  "CentOS Linux": "CentOS Linux",
  macOS: "macOS",
  "MS Windows": "Windows",
  "Red Hat Linux": "Red Hat Linux",
  "Ubuntu Linux": "Ubuntu Linux",
};

export const PLATFORM_LABEL_DISPLAY_ORDER = [
  "macOS",
  "All Linux",
  "CentOS Linux",
  "Red Hat Linux",
  "Ubuntu Linux",
  "MS Windows",
];

export const PLATFORM_LABEL_DISPLAY_TYPES: Record<string, string> = {
  "All Hosts": "all",
  "All Linux": "platform",
  "CentOS Linux": "platform",
  macOS: "platform",
  "MS Windows": "platform",
  "Red Hat Linux": "platform",
  "Ubuntu Linux": "platform",
};

export const PLATFORM_DROPDOWN_OPTIONS = [
  { label: "All", value: "" },
  { label: "Windows", value: "windows" },
  { label: "Linux", value: "linux" },
  { label: "macOS", value: "darwin" },
];

export const PLATFORM_NAME_TO_LABEL_NAME = {
  all: "",
  darwin: "macOS",
  windows: "MS Windows",
  linux: "All Linux",
};

export const HOSTS_SEARCH_BOX_PLACEHOLDER =
  "Search hostname, UUID, serial number, or IPv4";

export const HOSTS_SEARCH_BOX_TOOLTIP =
  "Search hosts by hostname, UUID, machine serial or private IP address";

export const VULNERABLE_DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "All software",
    value: false,
    helpText: "All software installed on your hosts.",
  },
  {
    disabled: false,
    label: "Vulnerable software",
    value: true,
    helpText:
      "All software installed on your hosts with detected vulnerabilities.",
  },
];

export const DEFAULT_CREATE_USER_ERRORS = {
  email: "",
  name: "",
  password: "",
  sso_enabled: null,
};

export const DEFAULT_CREATE_LABEL_ERRORS = {
  name: "",
};
