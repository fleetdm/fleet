import URL_PREFIX from "router/url_prefix";
import { IQueryPlatform } from "interfaces/query";
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
    query: `SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT "" AND filevault_status = 'on' LIMIT 1`,
    name: "Is Filevault enabled on macOS devices?",
    description:
      "Checks to make sure that the Filevault feature is enabled on macOS devices.",
    resolution:
      "Choose Apple menu > System Preferences, then click Security & Privacy. Click the FileVault tab. Click the Lock icon, then enter an administrator name and password. Click Turn On FileVault.",
    platform: "darwin",
  },
  {
    key: 2,
    query: "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1",
    name: "Is Gatekeeper enabled on macOS devices?",
    description:
      "Checks to make sure that the Gatekeeper feature is enabled on macOS devices. Gatekeeper tries to ensure only trusted software is run on a mac machine.",
    resolution:
      "On the failing device, run the following command in the Terminal app: /usr/sbin / spctl--master- enable",
    platform: "darwin",
  },
  {
    key: 3,
    query: "SELECT 1 FROM bitlocker_info WHERE protection_status = 1;",
    name: "Is disk encryption enabled on Windows devices?",
    description:
      "Checks to make sure that device encryption is enabled on Windows devices.",
    resolution:
      "Option 1: Select the Start button. Select Settings > Update & Security > Device encryption. If Device encryption doesn't appear, skip to Option 2. If device encryption is turned off, select Turn on. Option 2: Select the Start button. Under Windows System, select Control Panel. Select System and Security. Under BitLocker Drive Encryption, select Manage BitLocker. Select Turn on BitLocker and then follow the instructions.",
    platform: "windows",
  },
  {
    key: 4,
    query:
      "SELECT 1 FROM sip_config WHERE config_flag = 'sip' AND enabled = 1;",
    name: "Is System Integrity Protection (SIP) enabled on macOS devices?",
    description: "Checks to make sure that the SIP is enabled.",
    resolution:
      "On the failing device, run the following command in the Terminal app: /usr/sbin/spctl --master-enable",
    platform: "darwin",
  },
  {
    key: 5,
    query:
      "SELECT 1 FROM managed_policies WHERE domain = 'com.apple.loginwindow' AND name = 'com.apple.login.mcx.DisableAutoLoginClient' AND value = 1 LIMIT 1",
    name: "Is automatic login disabled on macOS devices?",
    description:
      "Required: You’re already enforcing a policy via Moble Device Management (MDM). Checks to make sure that the device user cannot log in to the device without a password. It’s good practice to have both this policy and the “Is Filevault enabled on macOS devices?” policy enabled.",
    resolution:
      "The following example profile includes a setting to disable automatic login: https://github.com/gregneagle/profiles/blob/fecc73d66fa17b6fa78b782904cb47cdc1913aeb/loginwindow.mobileconfig#L64-L65",
    platform: "darwin",
  },
  {
    key: 6,
    query:
      "SELECT 1 FROM managed_policies WHERE domain = 'com.apple.MCX' AND name = 'DisableGuestAccount' AND value = 0 LIMIT 1;",
    name: "Are guest users not activated on macOS devices?",
    description:
      "Required: You’re already enforcing a policy via Moble Device Management (MDM). Checks to make sure that guest accounts cannot be used to log in to the device without a password.",
    resolution:
      "The following example profile includes a setting to disable automatic login: https://github.com/gregneagle/profiles/blob/fecc73d66fa17b6fa78b782904cb47cdc1913aeb/loginwindow.mobileconfig#L68-L71",
    platform: "darwin",
  },
  {
    key: 7,
    query:
      "SELECT 1 FROM managed_policies WHERE domain = 'com.apple.Terminal' AND name = 'SecureKeyboardEntry' AND value=1 LIMIT 1;",
    name: "Is secure keyboard entry enabled on macOS devices?",
    description:
      "Required: You’re already enforcing a policy via Moble Device Management (MDM). Checks to make sure that the Secure Keyboard Entry setting is enabled.",
    resolution: "",
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

export const LOGGING_TYPE_OPTIONS = [
  { label: "Snapshot", value: "snapshot" },
  { label: "Differential", value: "differential" },
  {
    label: "Differential (Ignore Removals)",
    value: "differential_ignore_removals",
  },
];

export const MIN_OSQUERY_VERSION_OPTIONS = [
  { label: "All", value: "" },
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

const DEFAULT_POLICY_PLATFORM: IQueryPlatform = "";

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

export const DEFAULT_CREATE_USER_ERRORS = {
  email: "",
  name: "",
  password: "",
  sso_enabled: null,
};
