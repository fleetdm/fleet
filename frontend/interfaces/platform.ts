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
