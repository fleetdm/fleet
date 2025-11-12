const fleetMaintainedPackageTypes = ["dmg", "zip"] as const;
const unixPackageTypes = ["pkg", "deb", "rpm", "dmg", "zip", "tar.gz"] as const;
const windowsPackageTypes = ["msi", "exe"] as const;
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

export const isWindowsPackageType = (s: any): s is WindowsPackageType => {
  return windowsPackageTypes.includes(s);
};

export const isUnixPackageType = (s: any): s is UnixPackageType => {
  return unixPackageTypes.includes(s);
};

export const isFleetMaintainedPackageType = (
  s: any
): s is FleetMaintainedPackageType => {
  return fleetMaintainedPackageTypes.includes(s);
};

export const isIosIpadosPackageType = (s: any): s is IosIpadosPackageType => {
  return iosIpadosPackageTypes.includes(s);
};

export const isPackageType = (s: any): s is PackageType => {
  return packageTypes.includes(s);
};
