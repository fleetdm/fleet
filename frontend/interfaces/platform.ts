export type IOsqueryPlatform =
  | "darwin"
  | "macOS"
  | "windows"
  | "Windows"
  | "linux"
  | "Linux"
  | "freebsd"
  | "FreeBSD";

export type IPlatformString =
  | ""
  | "darwin"
  | "windows"
  | "linux"
  | "darwin,windows,linux"
  | "darwin,windows"
  | "darwin,linux"
  | "windows,linux";

export const SUPPORTED_PLATFORMS = ["darwin", "windows", "linux"] as const;

// TODO: revisit this approach pending resolution of https://github.com/fleetdm/fleet/issues/3555.
export const MACADMINS_EXTENSION_TABLES: Record<string, IOsqueryPlatform[]> = {
  file_lines: ["darwin", "linux", "windows"],
  filevault_users: ["darwin"],
  google_chrome_profiles: ["darwin", "linux", "windows"],
  macos_profiles: ["darwin"],
  mdm: ["darwin"],
  munki_info: ["darwin"],
  munki_install: ["darwin"],
  network_quality: ["darwin"],
  puppet_info: ["darwin", "linux", "windows"],
  puppet_logs: ["darwin", "linux", "windows"],
  puppet_state: ["darwin", "linux", "windows"],
  unified_log: ["darwin"],
};
