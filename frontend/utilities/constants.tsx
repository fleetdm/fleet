import URL_PREFIX from "router/url_prefix";
import { OsqueryPlatform } from "interfaces/platform";
import paths from "router/paths";
import { ISchedulableQuery } from "interfaces/schedulable_query";
import React from "react";

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

export const FREQUENCY_DROPDOWN_OPTIONS = [
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

export const GITHUB_NEW_ISSUE_LINK =
  "https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md";
export const SUPPORT_LINK = "https://fleetdm.com/support";

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
  { label: "5.10.2 +", value: "5.10.2" },
  { label: "5.9.1 +", value: "5.9.1" },
  { label: "5.8.2 +", value: "5.8.2" },
  { label: "5.8.1 +", value: "5.8.1" },
  { label: "5.7.0 +", value: "5.7.0" },
  { label: "5.6.0 +", value: "5.6.0" },
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

export const PLATFORM_DISPLAY_NAMES: Record<string, OsqueryPlatform> = {
  darwin: "macOS",
  macOS: "macOS",
  windows: "Windows",
  Windows: "Windows",
  linux: "Linux",
  Linux: "Linux",
  chrome: "ChromeOS",
  ChromeOS: "ChromeOS",
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
  chrome: "ChromeOS",
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
  chrome: "platform",
};

interface IPlatformDropdownOptions {
  label: "All" | "Windows" | "Linux" | "macOS" | "ChromeOS";
  value: "all" | "windows" | "linux" | "darwin" | "chrome" | "";
  path?: string;
}
export const PLATFORM_DROPDOWN_OPTIONS: IPlatformDropdownOptions[] = [
  { label: "All", value: "all", path: paths.DASHBOARD },
  { label: "macOS", value: "darwin", path: paths.DASHBOARD_MAC },
  { label: "Windows", value: "windows", path: paths.DASHBOARD_WINDOWS },
  { label: "Linux", value: "linux", path: paths.DASHBOARD_LINUX },
  { label: "ChromeOS", value: "chrome", path: paths.DASHBOARD_CHROME },
];

// Schedules does not support ChromeOS
export const SCHEDULE_PLATFORM_DROPDOWN_OPTIONS: IPlatformDropdownOptions[] = [
  { label: "All", value: "" }, // API empty string runs on all platforms
  { label: "macOS", value: "darwin" },
  { label: "Windows", value: "windows" },
  { label: "Linux", value: "linux" },
];

// Builtin label names returned from API
export const PLATFORM_NAME_TO_LABEL_NAME = {
  all: "",
  darwin: "macOS",
  windows: "MS Windows",
  linux: "All Linux",
  chrome: "chrome",
};

export const HOSTS_SEARCH_BOX_PLACEHOLDER =
  "Search name, hostname, UUID, serial number, or private IP address";

export const HOSTS_SEARCH_BOX_TOOLTIP =
  "Search hosts by name, hostname, UUID, serial number, or private IP address";

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

export const EXPLOITED_VULNERABILITIES_DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "All vulnerabilities",
    value: false,
    helpText: "All vulnerabilities detected on your hosts.",
  },
  {
    disabled: false,
    label: "Exploited vulnerabilities",
    value: true,
    helpText: "Vulnerabilities that have been actively exploited in the wild.",
  },
];

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
    <span>MDM was turned on manually. End users can turn MDM off.</span>
  ),
  Off: (
    <span>
      Hosts with MDM off don&apos;t receive macOS <br /> settings and macOS
      update encouragement.
    </span>
  ),
  Pending: (
    <span>
      Hosts ordered via Apple Business Manager <br /> (ABM). These will
      automatically enroll to Fleet <br /> and turn on MDM when they&apos;re
      unboxed.
    </span>
  ),
};

export const DEFAULT_CREATE_USER_ERRORS = {
  email: "",
  name: "",
  password: "",
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
  "enroll_secret_name",
  "detail_updated_at",
  "percent_disk_space_available",
  "gigs_disk_space_available",
  "team_name",
  "disk_encryption_enabled",
  "display_name", // Not rendered on my device page
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
];

export const HOST_OSQUERY_DATA = [
  "config_tls_refresh",
  "logger_tls_period",
  "distributed_interval",
];
