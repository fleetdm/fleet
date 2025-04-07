import React from "react";

import Icon from "components/Icon";
import { HOSTS_QUERY_PARAMS } from "services/entities/hosts";

export const MANAGE_HOSTS_PAGE_FILTER_KEYS = [
  "query",
  "team_id",
  "policy_id",
  "policy_response",
  "macos_settings",
  "software_id",
  "software_version_id",
  "software_title_id",
  HOSTS_QUERY_PARAMS.SOFTWARE_STATUS,
  "status",
  "mdm_id",
  "mdm_enrollment_status",
  "os_name",
  "os_version",
  "munki_issue_id",
  "low_disk_space",
  HOSTS_QUERY_PARAMS.OS_SETTINGS,
  HOSTS_QUERY_PARAMS.DISK_ENCRYPTION,
  "bootstrap_package",
] as const;

/*
 * These are the URL query params that are incompatible with non-status labels on the manage hosts page.
 * They should be stripped from the URL when a non-status label is selected.
 */
export const MANAGE_HOSTS_PAGE_LABEL_INCOMPATIBLE_QUERY_PARAMS = [
  "policy_id",
  "policy_response",
  "software_id",
  "software_version_id",
  "software_title_id",
  HOSTS_QUERY_PARAMS.SOFTWARE_STATUS,
  "bootstrap_package",
  "macos_settings",
  HOSTS_QUERY_PARAMS.OS_SETTINGS,
  HOSTS_QUERY_PARAMS.DISK_ENCRYPTION,
] as const;

// TODO: refactor to use this type as the location.query prop of the page
export type ManageHostsPageQueryParams = Record<
  | "page"
  | "order_key"
  | "order_direction"
  | typeof MANAGE_HOSTS_PAGE_FILTER_KEYS[number],
  string
>;

export const LABEL_SLUG_PREFIX = "labels/";

export const DEFAULT_SORT_HEADER = "display_name";
export const DEFAULT_SORT_DIRECTION = "asc";
export const DEFAULT_PAGE_SIZE = 50;
export const DEFAULT_PAGE_INDEX = 0;

export const hostSelectStatuses = [
  {
    disabled: false,
    label: "All hosts",
    value: "",
    helpText: "All hosts added to Fleet.",
  },
  {
    disabled: false,
    label: "Online hosts",
    value: "online",
    helpText: "Hosts that will respond to a live query.",
  },
  {
    disabled: false,
    label: "Offline hosts",
    value: "offline",
    helpText: "Hosts that won’t respond to a live query.",
  },
  {
    disabled: false,
    label: "Missing hosts",
    value: "missing",
    helpText: "Hosts that have been offline for 30 days or more.",
  },
  {
    disabled: false,
    label: "New hosts",
    value: "new",
    helpText: "Hosts added to Fleet in the last 24 hours.",
  },
];

export const OS_SETTINGS_FILTER_OPTIONS = [
  {
    disabled: false,
    label: "Verified",
    value: "verified",
  },
  {
    disabled: false,
    label: "Verifying",
    value: "verifying",
  },
  {
    disabled: false,
    label: "Pending",
    value: "pending",
  },
  {
    disabled: false,
    label: "Failed",
    value: "failed",
  },
];
