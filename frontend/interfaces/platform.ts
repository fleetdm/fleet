export type OsqueryPlatform =
  | "darwin"
  | "macOS"
  | "windows"
  | "Windows"
  | "linux"
  | "Linux"
  | "chrome"
  | "ChromeOS";

export type SupportedPlatform = "darwin" | "windows" | "linux" | "chrome";

export const SUPPORTED_PLATFORMS: SupportedPlatform[] = [
  "darwin",
  "windows",
  "linux",
  "chrome",
];
export type SelectedPlatform = SupportedPlatform | "all";

export type SelectedPlatformString =
  | ""
  | "darwin"
  | "windows"
  | "linux"
  | "chrome"
  | "darwin,windows,linux,chrome"
  | "darwin,windows,linux"
  | "darwin,linux,chrome"
  | "darwin,windows,chrome"
  | "windows,linux,chrome"
  | "darwin,windows"
  | "darwin,linux"
  | "darwin,chrome"
  | "windows,linux"
  | "windows,chrome"
  | "linux,chrome";

// TODO: revisit this approach pending resolution of https://github.com/fleetdm/fleet/issues/3555.
export const MACADMINS_EXTENSION_TABLES: Record<string, OsqueryPlatform[]> = {
  file_lines: ["darwin", "linux", "windows"],
  filevault_users: ["darwin"],
  google_chrome_profiles: ["darwin", "linux", "windows"],
  macos_profiles: ["darwin"],
  mdm: ["darwin"],
  munki_info: ["darwin"],
  munki_install: ["darwin"],
  // network_quality: ["darwin"], // TODO: add this table if/when it is incorporated into orbit
  puppet_info: ["darwin", "linux", "windows"],
  puppet_logs: ["darwin", "linux", "windows"],
  puppet_state: ["darwin", "linux", "windows"],
  macadmins_unified_log: ["darwin"],
};
