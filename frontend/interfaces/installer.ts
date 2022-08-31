export type IInstallerType = "pkg" | "msi" | "rpm" | "deb";

export type IInstallerPlatform =
  | "Windows"
  | "macOS"
  | "Linux (RPM)"
  | "Linux (deb)";

export const INSTALLER_TYPE_BY_PLATFORM: Record<
  IInstallerPlatform,
  IInstallerType
> = {
  macOS: "pkg",
  Windows: "msi",
  "Linux (RPM)": "rpm",
  "Linux (deb)": "deb",
} as const;

export const INSTALLER_PLATFORM_BY_TYPE: Record<
  IInstallerType,
  IInstallerPlatform
> = {
  pkg: "macOS",
  msi: "Windows",
  rpm: "Linux (RPM)",
  deb: "Linux (deb)",
} as const;
