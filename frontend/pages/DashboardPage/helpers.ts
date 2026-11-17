import { IHostSummaryPlatforms } from "interfaces/host_summary";
import { isLinuxLike } from "interfaces/platform";
import paths from "router/paths";

import { IHostPlatformCounts } from "./cards/HostsEnrolledCard";

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

/** Host counts for the enrolled hosts card. Every Linux distribution adds to `linux`. */
export const getPlatformHostCounts = (
  platforms?: IHostSummaryPlatforms[] | null
): IHostPlatformCounts => {
  const counts: IHostPlatformCounts = {
    darwin: 0,
    windows: 0,
    linux: 0,
    chrome: 0,
    ios: 0,
    ipados: 0,
    android: 0,
  };
  (platforms ?? []).forEach((item) => {
    if (isLinuxLike(item.platform)) {
      counts.linux += item.hosts_count || 0;
    } else if (item.platform in counts) {
      counts[item.platform as keyof IHostPlatformCounts] =
        item.hosts_count || 0;
    }
  });
  return counts;
};
