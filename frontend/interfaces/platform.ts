export const APPLE_PLATFORM_DISPLAY_NAMES = {
  darwin: "macOS",
  ios: "iOS",
  ipados: "iPadOS",
} as const;

export type ApplePlatform = keyof typeof APPLE_PLATFORM_DISPLAY_NAMES;
export type AppleDisplayPlatform = typeof APPLE_PLATFORM_DISPLAY_NAMES[keyof typeof APPLE_PLATFORM_DISPLAY_NAMES];

export const PLATFORM_DISPLAY_NAMES = {
  windows: "Windows",
  linux: "Linux",
  chrome: "ChromeOS",
  ...APPLE_PLATFORM_DISPLAY_NAMES,
} as const;

export type Platform = keyof typeof PLATFORM_DISPLAY_NAMES;
export type DisplayPlatform = typeof PLATFORM_DISPLAY_NAMES[keyof typeof PLATFORM_DISPLAY_NAMES];
export type QueryableDisplayPlatform = Exclude<
  DisplayPlatform,
  "iOS" | "iPadOS"
>;
export type QueryablePlatform = Exclude<Platform, "ios" | "ipados">;

export const SUPPORTED_PLATFORMS: QueryablePlatform[] = [
  "darwin",
  "windows",
  "linux",
  "chrome",
];

// TODO - add "iOS" and "iPadOS" once we support them
export const VULN_SUPPORTED_PLATFORMS: Platform[] = ["darwin", "windows"];

export type SelectedPlatform = QueryablePlatform | "all";

export type SelectedPlatformString =
  | ""
  | QueryablePlatform
  | `${QueryablePlatform},${QueryablePlatform}`
  | `${QueryablePlatform},${QueryablePlatform},${QueryablePlatform}`
  | `${QueryablePlatform},${QueryablePlatform},${QueryablePlatform},${QueryablePlatform}`;

// TODO: revisit this approach pending resolution of https://github.com/fleetdm/fleet/issues/3555.
export const MACADMINS_EXTENSION_TABLES: Record<string, QueryablePlatform[]> = {
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
  "tuxedo",
] as const;

export const HOST_APPLE_PLATFORMS = ["darwin", "ios", "ipados"] as const;

export type HostPlatform =
  | typeof HOST_LINUX_PLATFORMS[number]
  | typeof HOST_APPLE_PLATFORMS[number]
  | "windows"
  | "chrome";

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

export const isAppleDevice = (platform: string) => {
  return HOST_APPLE_PLATFORMS.includes(
    platform as typeof HOST_APPLE_PLATFORMS[number]
  );
};

export const isIPadOrIPhone = (platform: string | HostPlatform) =>
  ["ios", "ipados"].includes(platform);
