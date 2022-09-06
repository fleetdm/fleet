export interface IOperatingSystemVersion {
  os_id: number;
  name: string;
  name_only: string;
  version: string;
  platform: string;
  hosts_count: number;
}

export const OS_VENDOR_BY_PLATFORM: Record<string, string> = {
  darwin: "Apple",
  windows: "Microsoft",
} as const;

export const OS_END_OF_LIFE_LINK_BY_PLATFORM: Record<string, string> = {
  darwin: "https://endoflife.date/macos",
  windows: "https://endoflife.date/windows",
} as const;

export const formatOperatingSystemDisplayName = (name: string) => {
  let displayName = name;
  if (displayName.startsWith("Microsoft Windows")) {
    displayName = displayName.replace("Microsoft Windows", "Windows");
  }
  return displayName;
};
