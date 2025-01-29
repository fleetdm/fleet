import paths from "router/paths";

import {
  PlatformLabelOptions,
  PlatformValueOptions,
} from "utilities/constants";

interface IPlatformDropdownOptions {
  label: PlatformLabelOptions;
  value: PlatformValueOptions;
  path: string;
}

/** Select platform */
export const PLATFORM_DROPDOWN_OPTIONS: IPlatformDropdownOptions[] = [
  { label: "All", value: "all", path: paths.DASHBOARD },
  { label: "macOS", value: "darwin", path: paths.DASHBOARD_MAC },
  { label: "Windows", value: "windows", path: paths.DASHBOARD_WINDOWS },
  { label: "Linux", value: "linux", path: paths.DASHBOARD_LINUX },
  { label: "ChromeOS", value: "chrome", path: paths.DASHBOARD_CHROME },
  { label: "iOS", value: "ios", path: paths.DASHBOARD_IOS },
  { label: "iPadOS", value: "ipados", path: paths.DASHBOARD_IPADOS },
];

/** Selected platform value mapped to built in label name */
export const PLATFORM_NAME_TO_LABEL_NAME = {
  darwin: "macOS",
  windows: "MS Windows",
  linux: "All Linux",
  chrome: "chrome",
  ios: "iOS",
  ipados: "iPadOS",
};

/** Premium feature, Gb must be set between 1-100 */
export const LOW_DISK_SPACE_GB = 32;
