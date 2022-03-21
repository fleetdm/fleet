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

// TODO: How do we want to handle checking platform compatibility for extension tables? See template
// "MDM enrolled" policy for example of how this can be an issue where tables are not included in
// osquery_tables.json. One approach would be to maintain a separate constant that lists extension
// tables as below.
export const EXTENSION_TABLES: Record<string, IOsqueryPlatform[]> = {
  mdm: ["darwin"],
};
