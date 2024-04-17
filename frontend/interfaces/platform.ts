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
  | SupportedPlatform
  | `${SupportedPlatform},${SupportedPlatform}`
  | `${SupportedPlatform},${SupportedPlatform},${SupportedPlatform}`
  | `${SupportedPlatform},${SupportedPlatform},${SupportedPlatform},${SupportedPlatform}`;

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

/**
 * Host Linux OSs as defined by the Fleet server.
 *
 * @see https://github.com/fleetdm/fleet/blob/5a21e2cfb029053ddad0508869eb9f1f23997bf2/server/fleet/hosts.go#L780
 */
export const HOST_LINUX_PLATFORMS = [
  "linux",
  "ubuntu",
  "debian",
  "rhel",
  "centos",
  "sles",
  "kali",
  "gentoo",
  "amzn",
  "pop",
  "arch",
  "linuxmint",
  "void",
  "nixos",
  "endeavouros",
  "manjaro",
  "opensuse-leap",
  "opensuse-tumbleweed",
] as const;

/**
 * Checks if the provided platform is a Linux-like OS. We can recieve many
 * different types of host platforms so we need a check that will cover all
 * the possible Linux-like platform values.
 */
export const isLinuxLike = (platform: string) => {
  return HOST_LINUX_PLATFORMS.includes(
    platform as typeof HOST_LINUX_PLATFORMS[number]
  );
};
