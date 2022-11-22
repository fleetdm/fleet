export const LABEL_SLUG_PREFIX = "labels/";

export const DEFAULT_SORT_HEADER = "display_name";
export const DEFAULT_SORT_DIRECTION = "asc";
export const DEFAULT_PAGE_SIZE = 20;

export const HOST_SELECT_STATUSES = [
  {
    disabled: false,
    label: "All hosts",
    value: "",
    helpText: "All hosts that have been enrolled to Fleet.",
  },
  {
    disabled: false,
    label: "Online hosts",
    value: "online",
    helpText: "Hosts that have recently checked in to Fleet.",
  },
  {
    disabled: false,
    label: "Offline hosts",
    value: "offline",
    helpText: "Hosts that have not checked in to Fleet recently.",
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
    helpText: "Hosts that have been enrolled to Fleet in the last 24 hours.",
  },
];
