const fleetMaintainedPackageTypes = ["dmg", "zip"] as const;
const unixPackageTypes = ["pkg", "deb", "rpm", "dmg", "zip", "tar.gz"] as const;
const windowsPackageTypes = ["msi", "exe", "zip"] as const;
const scriptOnlyPackageTypes = ["sh", "ps1"] as const;
const iosIpadosPackageTypes = ["ipa"] as const;
export const packageTypes = [
  ...unixPackageTypes,
  ...windowsPackageTypes,
  ...scriptOnlyPackageTypes,
  ...iosIpadosPackageTypes,
] as const;

export type WindowsPackageType = typeof windowsPackageTypes[number];
export type UnixPackageType = typeof unixPackageTypes[number];
export type FleetMaintainedPackageType = typeof fleetMaintainedPackageTypes[number];
export type ScriptOnlyPackageType = typeof scriptOnlyPackageTypes[number];
export type IosIpadosPackageType = typeof iosIpadosPackageTypes[number];
export type PackageType =
  | WindowsPackageType
  | UnixPackageType
  | FleetMaintainedPackageType
  | ScriptOnlyPackageType
  | IosIpadosPackageType;

export const isWindowsPackageType = (s: unknown): s is WindowsPackageType => {
  return (
    typeof s === "string" &&
    windowsPackageTypes.includes(s as WindowsPackageType)
  );
};

export const isUnixPackageType = (s: unknown): s is UnixPackageType => {
  return (
    typeof s === "string" && unixPackageTypes.includes(s as UnixPackageType)
  );
};

export const isFleetMaintainedPackageType = (
  s: unknown
): s is FleetMaintainedPackageType => {
  return (
    typeof s === "string" &&
    fleetMaintainedPackageTypes.includes(s as FleetMaintainedPackageType)
  );
};

export const isIosIpadosPackageType = (
  s: unknown
): s is IosIpadosPackageType => {
  return (
    typeof s === "string" &&
    iosIpadosPackageTypes.includes(s as IosIpadosPackageType)
  );
};

export const isPackageType = (s: unknown): s is PackageType => {
  return typeof s === "string" && packageTypes.includes(s as PackageType);
};
