import URL_PREFIX from "router/url_prefix";
import { DisplayPlatform, Platform } from "interfaces/platform";
import { ISchedulableQuery } from "interfaces/schedulable_query";
import React from "react";
import { IDropdownOption } from "interfaces/dropdownOption";
import { IconNames } from "components/icons";

const { origin } = global.window.location;
export const BASE_URL = `${origin}${URL_PREFIX}/api`;

export enum PolicyResponse {
  PASSING = "passing",
  FAILING = "failing",
}

export const DEFAULT_GRAVATAR_LINK =
  "https://fleetdm.com/images/permanent/icon-avatar-default-transparent-64x64%402x.png";

export const DEFAULT_GRAVATAR_LINK_DARK =
  "https://fleetdm.com/images/permanent/icon-avatar-default-dark-24x24%402x.png";

export const DEFAULT_GRAVATAR_LINK_FALLBACK =
  "/assets/images/icon-avatar-default-transparent-64x64%402x.png";

export const DEFAULT_GRAVATAR_LINK_DARK_FALLBACK =
  "/assets/images/icon-avatar-default-dark-24x24%402x.png";

export const ACTIVITY_EXPIRY_WINDOW_DROPDOWN_OPTIONS: IDropdownOption[] = [
  { value: 30, label: "30 days" },
  { value: 60, label: "60 days" },
  { value: 90, label: "90 days" },
];

export const FREQUENCY_DROPDOWN_OPTIONS: IDropdownOption[] = [
  { value: 0, label: "Never" },
  { value: 300, label: "Every 5 minutes" },
  { value: 600, label: "Every 10 minutes" },
  { value: 900, label: "Every 15 minutes" },
  { value: 1800, label: "Every 30 minutes" },
  { value: 3600, label: "Every hour" },
  { value: 21600, label: "Every 6 hours" },
  { value: 43200, label: "Every 12 hours" },
  { value: 86400, label: "Every day" },
  { value: 604800, label: "Every week" },
];
export const HOST_STATUS_WEBHOOK_HOST_PERCENTAGE_DROPDOWN_OPTIONS: IDropdownOption[] = [
  { label: "1%", value: 1 },
  { label: "5%", value: 5 },
  { label: "10%", value: 10 },
  { label: "25%", value: 25 },
];

export const HOST_STATUS_WEBHOOK_WINDOW_DROPDOWN_OPTIONS: IDropdownOption[] = [
  { label: "1 day", value: 1 },
  { label: "3 days", value: 3 },
  { label: "7 days", value: 7 },
  { label: "14 days", value: 14 },
];

export const GITHUB_NEW_ISSUE_LINK =
  "https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md";

export const FLEET_WEBSITE_URL = "https://fleetdm.com";

export const SUPPORT_LINK = `${FLEET_WEBSITE_URL}/support`;

export const CONTACT_FLEET_LINK = `${FLEET_WEBSITE_URL}/contact`;

export const LEARN_MORE_ABOUT_BASE_LINK = `${FLEET_WEBSITE_URL}/learn-more-about`;

/**  July 28, 2016 is the date of the initial commit to fleet/fleet. */
export const INITIAL_FLEET_DATE = "2016-07-28T00:00:00Z";

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
  { label: "5.15.0 +", value: "5.15.0" },
  { label: "5.14.1 +", value: "5.14.1" },
  { label: "5.13.1 +", value: "5.13.1" },
  { label: "5.12.2 +", value: "5.12.2" },
  { label: "5.12.1 +", value: "5.12.1" },
  { label: "5.11.0 +", value: "5.11.0" },
  { label: "5.10.2 +", value: "5.10.2" },
  { label: "5.9.1 +", value: "5.9.1" },
  { label: "5.8.2 +", value: "5.8.2" },
  { label: "5.8.1 +", value: "5.8.1" },
  { label: "5.7.0 +", value: "5.7.0" },
  { label: "5.6.0 +", value: "5.6.0" },
  { label: "5.5.1 +", value: "5.5.1" },
  { label: "5.4.0 +", value: "5.4.0" },
  { label: "5.3.0 +", value: "5.3.0" },
  { label: "5.2.3 +", value: "5.2.3" },
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

export const LIVE_POLICY_STEPS = {
  1: "EDITOR",
  2: "TARGETS",
  3: "RUN",
};

export const LIVE_QUERY_STEPS = {
  1: "TARGETS",
  2: "RUN",
};

export const DEFAULT_QUERY: ISchedulableQuery = {
  description: "",
  name: "",
  query: "SELECT * FROM osquery_info;",
  id: 0,
  interval: 0,
  observer_can_run: false,
  discard_data: false,
  platform: "",
  min_osquery_version: "",
  automations_enabled: false,
  logging: "snapshot",
  author_name: "",
  updated_at: "",
  created_at: "",
  saved: false,
  author_id: 0,
  packs: [],
  team_id: 0,
  author_email: "",
  stats: {},
  editingExistingQuery: false,
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

const PLATFORM_LABEL_NAMES_FROM_API = [
  "All Hosts",
  "All Linux",
  "CentOS Linux",
  "macOS",
  "MS Windows",
  "Red Hat Linux",
  "Ubuntu Linux",
  "chrome",
  "iOS",
  "iPadOS",
] as const;

export type PlatformLabelNameFromAPI = typeof PLATFORM_LABEL_NAMES_FROM_API[number];

export const isPlatformLabelNameFromAPI = (
  s: string
): s is PlatformLabelNameFromAPI => {
  return PLATFORM_LABEL_NAMES_FROM_API.includes(s as PlatformLabelNameFromAPI);
};

export const PLATFORM_DISPLAY_NAMES: Record<string, DisplayPlatform> = {
  darwin: "macOS",
  macOS: "macOS",
  windows: "Windows",
  Windows: "Windows",
  linux: "Linux",
  Linux: "Linux",
  chrome: "ChromeOS",
  ChromeOS: "ChromeOS",
  ios: "iOS",
  ipados: "iPadOS",
} as const;

// as returned by the TARGETS API; based on display_text
export const PLATFORM_LABEL_DISPLAY_NAMES: Record<
  PlatformLabelNameFromAPI,
  string
> = {
  "All Hosts": "All hosts",
  "All Linux": "Linux",
  "CentOS Linux": "CentOS Linux",
  macOS: "macOS",
  "MS Windows": "Windows",
  "Red Hat Linux": "Red Hat Linux",
  "Ubuntu Linux": "Ubuntu Linux",
  chrome: "ChromeOS",
  iOS: "iOS",
  iPadOS: "iPadOS",
} as const;

export const PLATFORM_LABEL_DISPLAY_TYPES: Record<
  PlatformLabelNameFromAPI,
  string
> = {
  "All Hosts": "all",
  "All Linux": "platform",
  "CentOS Linux": "platform",
  macOS: "platform",
  "MS Windows": "platform",
  "Red Hat Linux": "platform",
  "Ubuntu Linux": "platform",
  chrome: "platform",
  iOS: "platform",
  iPadOS: "platform",
} as const;

// For some builtin labels, display different strings than what API returns
export const LABEL_DISPLAY_MAP: Partial<
  Record<PlatformLabelNameFromAPI, string>
> = {
  "All Hosts": "All hosts",
  "All Linux": "Linux",
  chrome: "ChromeOS",
  "MS Windows": "Windows",
};

export const PLATFORM_TYPE_ICONS: Record<
  Extract<
    PlatformLabelNameFromAPI,
    "All Linux" | "macOS" | "MS Windows" | "chrome" | "iOS" | "iPadOS"
  >,
  IconNames
> = {
  "All Linux": "linux",
  macOS: "darwin",
  "MS Windows": "windows",
  chrome: "chrome",
  iOS: "iOS",
  iPadOS: "iPadOS",
} as const;

export const hasPlatformTypeIcon = (
  s: string
): s is Extract<
  PlatformLabelNameFromAPI,
  "All Linux" | "macOS" | "MS Windows" | "chrome" | "iOS" | "iPadOS"
> => {
  return !!PLATFORM_TYPE_ICONS[s as keyof typeof PLATFORM_TYPE_ICONS];
};

export type PlatformLabelOptions = DisplayPlatform | "All";

export type PlatformValueOptions = Platform | "all";

/** Scheduled queries do not support ChromeOS, iOS, or iPadOS */
interface ISchedulePlatformDropdownOptions {
  label: Exclude<PlatformLabelOptions, "ChromeOS" | "iOS" | "iPadOS">;
  value: Exclude<PlatformValueOptions, "chrome" | "ios" | "ipados"> | "";
}

export const SCHEDULE_PLATFORM_DROPDOWN_OPTIONS: ISchedulePlatformDropdownOptions[] = [
  { label: "All", value: "" }, // API empty string runs on all platforms
  { label: "macOS", value: "darwin" },
  { label: "Windows", value: "windows" },
  { label: "Linux", value: "linux" },
];

export const HOSTS_SEARCH_BOX_PLACEHOLDER =
  "Search name, hostname, UUID, serial number, or private IP address";

export const HOSTS_SEARCH_BOX_TOOLTIP =
  "Search hosts by name, hostname, UUID, serial number, or private IP address";

export const VULNERABILITIES_SEARCH_BOX_TOOLTIP =
  'To search for an exact CVE, surround the string in double quotes (e.g. "CVE-2024-1234")';

// Keys from API
export const MDM_STATUS_TOOLTIP: Record<string, string | React.ReactNode> = {
  "On (automatic)": (
    <span>
      MDM was turned on automatically using Apple Automated Device Enrollment
      (DEP), Windows Autopilot, or Windows Azure AD Join. Administrators can
      block end users from turning MDM off.
    </span>
  ),
  "On (manual)": (
    <span>
      MDM was turned on manually (macOS), or hosts were automatically migrated
      with fleetd (Windows). End users can turn MDM off.
    </span>
  ),
  Off: undefined, // no tooltip specified
  Pending: (
    <span>
      Hosts ordered via Apple Business Manager <br /> (ABM). These will
      automatically enroll to Fleet <br /> and turn on MDM when they&apos;re
      unboxed.
    </span>
  ),
};

export const BATTERY_TOOLTIP: Record<string, string | React.ReactNode> = {
  Normal: (
    <span>
      Current maximum capacity is at least
      <br />
      80% of its designed capacity and the
      <br />
      cycle count is below 1000.
    </span>
  ),
  "Service recommended": (
    <span>
      Current maximum capacity has fallen
      <br />
      below 80% of its designed capacity
      <br />
      or the cycle count has reached 1000.
    </span>
  ),
};

export const DEFAULT_USER_FORM_ERRORS = {
  email: null,
  name: null,
  password: null,
  sso_enabled: null,
};

/** Must pass agent options config as empty object */
export const EMPTY_AGENT_OPTIONS = {
  config: {},
};

export const DEFAULT_EMPTY_CELL_VALUE = "---";

export const DOCUMENT_TITLE_SUFFIX = "Fleet";

export const HOST_SUMMARY_DATA = [
  "id",
  "status",
  "issues",
  "memory",
  "cpu_type",
  "platform",
  "os_version",
  "osquery_version",
  "orbit_version",
  "fleet_desktop_version",
  "enroll_secret_name",
  "detail_updated_at",
  "percent_disk_space_available",
  "gigs_disk_space_available",
  "team_name",
  "disk_encryption_enabled",
  "display_name", // Not rendered on my device page
  "maintenance_window", // Not rendered on my device page
];

export const HOST_ABOUT_DATA = [
  "seen_time",
  "uptime",
  "last_enrolled_at",
  "hardware_model",
  "hardware_serial",
  "primary_ip",
  "public_ip",
  "geolocation",
  "batteries",
  "detail_updated_at",
  "last_restarted_at",
  "platform",
];

export const HOST_OSQUERY_DATA = [
  "config_tls_refresh",
  "logger_tls_period",
  "distributed_interval",
];

export const DEFAULT_USE_QUERY_OPTIONS = {
  retry: 3,
  refetchOnWindowFocus: false,
};

export const INVALID_PLATFORMS_REASON =
  "query payload verification: query's platform must be a comma-separated list of 'darwin', 'linux', 'windows', and/or 'chrome' in a single string";

export const INVALID_PLATFORMS_FLASH_MESSAGE =
  "Couldn't save query. Please update platforms and try again.";

export const DATE_FNS_FORMAT_STRINGS = {
  dateAtTime: "E, MMM d 'at' p",
  hoursAndMinutes: "HH:mm",
};
