import paths from "router/paths";

/** Select platform */
export const PLATFORM_DROPDOWN_OPTIONS = [
  { label: "All", value: "all", path: paths.DASHBOARD },
  { label: "macOS", value: "darwin", path: paths.DASHBOARD_MAC },
  { label: "Windows", value: "windows", path: paths.DASHBOARD_WINDOWS },
  { label: "Linux", value: "linux", path: paths.DASHBOARD_LINUX },
  { label: "ChromeOS", value: "chrome", path: paths.DASHBOARD_CHROME },
  { label: "iOS", value: "ios", path: paths.DASHBOARD_IOS },
  { label: "iPadOS", value: "ipados", path: paths.DASHBOARD_IPADOS },
  { label: "Android", value: "android", path: paths.DASHBOARD_ANDROID },
] as const;

/** Premium feature, Gb must be set between 1-100 */
export const LOW_DISK_SPACE_GB = 32;
