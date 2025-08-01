import { startCase } from "lodash";
import PropTypes from "prop-types";

import { IconNames } from "components/icons";

import { HOST_APPLE_PLATFORMS, Platform } from "./platform";
import vulnerabilityInterface from "./vulnerability";
import { ILabelSoftwareTitle } from "./label";

export default PropTypes.shape({
  type: PropTypes.string,
  name: PropTypes.string,
  version: PropTypes.string,
  source: PropTypes.string,
  id: PropTypes.number,
  vulnerabilities: PropTypes.arrayOf(vulnerabilityInterface),
});

export interface ISoftwareResponse {
  counts_updated_at: string;
  software: ISoftware[];
}

export interface ISoftwareCountResponse {
  count: number;
}

export interface IGetSoftwareByIdResponse {
  software: ISoftware;
}

// TODO: old software interface. replaced with ISoftwareVersion
// check to see if we still need this.
export interface ISoftware {
  id: number;
  name: string; // e.g., "Figma.app"
  version: string; // e.g., "2.1.11"
  bundle_identifier?: string | null; // e.g., "com.figma.Desktop"
  source: string; // "apps" | "ipados_apps" | "ios_apps" | "programs" | "rpm_packages" | "deb_packages" | ?
  generated_cpe: string;
  vulnerabilities: ISoftwareVulnerability[] | null;
  hosts_count?: number;
  last_opened_at?: string | null; // e.g., "2021-08-18T15:11:35Z”
  installed_paths?: string[];
  browser?: string;
  vendor?: string;
}

export type IVulnerabilitySoftware = Omit<ISoftware, "vulnerabilities"> & {
  resolved_in_version: string;
};

export interface ISoftwareTitleVersion {
  id: number;
  version: string;
  vulnerabilities: string[] | null; // TODO: does this return null or is it omitted?
  hosts_count?: number;
}

export interface ISoftwareInstallPolicy {
  id: number;
  name: string;
}

export type SoftwareCategory =
  | "Browsers"
  | "Communication"
  | "Developer tools"
  | "Productivity";

export interface ISoftwarePackageStatus {
  installed: number;
  pending_install: number;
  failed_install: number;
  pending_uninstall: number;
  failed_uninstall: number;
}

export interface ISoftwareAppStoreAppStatus {
  installed: number;
  pending: number;
  failed: number;
}

export interface ISoftwarePackage {
  name: string;
  title_id: number;
  url: string;
  version: string;
  uploaded_at: string;
  install_script: string;
  uninstall_script: string;
  pre_install_query?: string;
  post_install_script?: string;
  automatic_install?: boolean; // POST only
  self_service: boolean;
  icon_url: string | null;
  status: ISoftwarePackageStatus;
  automatic_install_policies?: ISoftwareInstallPolicy[] | null;
  install_during_setup?: boolean;
  labels_include_any: ILabelSoftwareTitle[] | null;
  labels_exclude_any: ILabelSoftwareTitle[] | null;
  categories?: SoftwareCategory[];
  fleet_maintained_app_id?: number | null;
  hash_sha256?: string | null;
}

export const isSoftwarePackage = (
  data: ISoftwarePackage | IAppStoreApp
): data is ISoftwarePackage =>
  (data as ISoftwarePackage).install_script !== undefined;

export interface IAppStoreApp {
  name: string;
  app_store_id: number;
  latest_version: string;
  created_at: string;
  icon_url: string;
  self_service: boolean;
  platform: typeof HOST_APPLE_PLATFORMS[number];
  status: ISoftwareAppStoreAppStatus;
  install_during_setup?: boolean;
  automatic_install_policies?: ISoftwareInstallPolicy[] | null;
  automatic_install?: boolean;
  last_install?: IAppLastInstall | null;
  last_uninstall?: {
    script_execution_id: string;
    uninstalled_at: string;
  } | null;
  version?: string;
  labels_include_any: ILabelSoftwareTitle[] | null;
  labels_exclude_any: ILabelSoftwareTitle[] | null;
  categories?: SoftwareCategory[];
}

export interface ISoftwareTitle {
  id: number;
  name: string;
  versions_count: number;
  source: SoftwareSource;
  hosts_count: number;
  versions: ISoftwareTitleVersion[] | null;
  software_package: ISoftwarePackage | null;
  app_store_app: IAppStoreApp | null;
  browser?: BrowserType;
}

export interface ISoftwareTitleDetails {
  id: number;
  name: string;
  software_package: ISoftwarePackage | null;
  app_store_app: IAppStoreApp | null;
  source: SoftwareSource;
  hosts_count: number;
  versions: ISoftwareTitleVersion[] | null;
  counts_updated_at?: string;
  bundle_identifier?: string;
  browser?: BrowserType;
  versions_count?: number;
}

export interface ISoftwareVulnerability {
  cve: string;
  details_link: string;
  cvss_score?: number | null;
  epss_probability?: number | null;
  cisa_known_exploit?: boolean | null;
  cve_published?: string | null;
  cve_description?: string | null;
  resolved_in_version?: string | null;
  created_at?: string | null;
}

export interface ISoftwareVersion {
  id: number;
  name: string; // e.g., "Figma.app"
  version: string; // e.g., "2.1.11"
  bundle_identifier?: string; // e.g., "com.figma.Desktop"
  source: SoftwareSource;
  browser: BrowserType;
  release: string; // TODO: on software/verions/:id?
  vendor: string;
  arch: string; // e.g., "x86_64" // TODO: on software/verions/:id?
  generated_cpe: string;
  vulnerabilities: ISoftwareVulnerability[] | null;
  hosts_count?: number;
}

export const SOURCE_TYPE_CONVERSION = {
  apt_sources: "Package (APT)",
  deb_packages: "Package (deb)",
  portage_packages: "Package (Portage)",
  rpm_packages: "Package (RPM)",
  yum_sources: "Package (YUM)",
  npm_packages: "Package (NPM)",
  atom_packages: "Package (Atom)", // Atom packages were removed from software inventory. Mapping is maintained for backwards compatibility. (2023-12-04)
  python_packages: "Package (Python)",
  tgz_packages: "Package (tar)",
  apps: "Application (macOS)",
  ios_apps: "Application (iOS)",
  ipados_apps: "Application (iPadOS)",
  chrome_extensions: "Browser plugin", // chrome_extensions can include any chrome-based browser (e.g., edge), so we rely instead on the `browser` field computed by Fleet server and fallback to this value if it is not present.
  firefox_addons: "Browser plugin (Firefox)",
  safari_extensions: "Browser plugin (Safari)",
  homebrew_packages: "Package (Homebrew)",
  programs: "Program (Windows)",
  ie_extensions: "Browser plugin (IE)",
  chocolatey_packages: "Package (Chocolatey)",
  pkg_packages: "Package (pkg)",
  vscode_extensions: "IDE extension (VS Code)",
} as const;

export type SoftwareSource = keyof typeof SOURCE_TYPE_CONVERSION;

/** Map installable software source to platform  */
export const INSTALLABLE_SOURCE_PLATFORM_CONVERSION = {
  apt_sources: "linux",
  deb_packages: "linux",
  portage_packages: "linux",
  rpm_packages: "linux",
  yum_sources: "linux",
  tgz_packages: "linux",
  npm_packages: null,
  atom_packages: null,
  python_packages: null,
  apps: "darwin",
  ios_apps: "ios",
  ipados_apps: "ipados",
  chrome_extensions: null,
  firefox_addons: null,
  safari_extensions: null,
  homebrew_packages: "darwin",
  programs: "windows",
  ie_extensions: null,
  chocolatey_packages: "windows",
  pkg_packages: "darwin",
  vscode_extensions: null,
} as const;

export type InstallableSoftwareSource = keyof typeof INSTALLABLE_SOURCE_PLATFORM_CONVERSION;

const BROWSER_TYPE_CONVERSION = {
  chrome: "Chrome",
  chromium: "Chromium",
  opera: "Opera",
  yandex: "Yandex",
  brave: "Brave",
  edge: "Edge",
  edge_beta: "Edge Beta",
} as const;

export type BrowserType = keyof typeof BROWSER_TYPE_CONVERSION;

export const formatSoftwareType = ({
  source,
  browser,
}: {
  source: SoftwareSource;
  browser?: BrowserType;
}) => {
  let type: string = SOURCE_TYPE_CONVERSION[source] || "Unknown";
  if (browser) {
    type = `Browser plugin (${
      BROWSER_TYPE_CONVERSION[browser] || startCase(browser)
    })`;
  }
  return type;
};

/**
 * This list comprises all possible states of software install operations.
 */
export const SOFTWARE_UNINSTALL_STATUSES = [
  "uninstalled",
  "pending_uninstall",
  "failed_uninstall",
] as const;

export type SoftwareUninstallStatus = typeof SOFTWARE_UNINSTALL_STATUSES[number];

export const SOFTWARE_INSTALL_STATUSES = [
  "installed",
  "pending_install",
  "failed_install",
  ...SOFTWARE_UNINSTALL_STATUSES,
] as const;

/*
 * SoftwareInstallStatus represents the possible states of software install operations.
 */
export type SoftwareInstallStatus = typeof SOFTWARE_INSTALL_STATUSES[number];

export const isValidSoftwareInstallStatus = (
  s: string | undefined | null
): s is SoftwareInstallStatus =>
  !!s && SOFTWARE_INSTALL_STATUSES.includes(s as SoftwareInstallStatus);

export const SOFTWARE_AGGREGATE_STATUSES = [
  "installed",
  "pending",
  "failed",
] as const;

export type SoftwareAggregateStatus = typeof SOFTWARE_AGGREGATE_STATUSES[number];

export const isValidSoftwareAggregateStatus = (
  s: string | undefined | null
): s is SoftwareAggregateStatus =>
  !!s && SOFTWARE_AGGREGATE_STATUSES.includes(s as SoftwareAggregateStatus);

export const isSoftwareUninstallStatus = (
  s: string | undefined | null
): s is SoftwareUninstallStatus =>
  !!s && SOFTWARE_UNINSTALL_STATUSES.includes(s as SoftwareUninstallStatus);

// not a typeguard, as above 2 functions are
export const isPendingStatus = (s: string | undefined | null) =>
  ["pending_install", "pending_uninstall"].includes(s || "");

export const resolveUninstallStatus = (
  activityStatus?: string
): SoftwareUninstallStatus => {
  let resolvedStatus = activityStatus;
  if (resolvedStatus === "pending") {
    resolvedStatus = "pending_uninstall";
  }
  if (resolvedStatus === "failed") {
    resolvedStatus = "failed_uninstall";
  }
  if (!isSoftwareUninstallStatus(resolvedStatus)) {
    console.warn(
      `Unexpected uninstall status "${activityStatus}" for activity. Defaulting to "pending_uninstall".`
    );
    resolvedStatus = "pending_uninstall";
  }
  return resolvedStatus as SoftwareUninstallStatus;
};

/**
 * ISoftwareInstallResult is the shape of a software install result object
 * returned by the Fleet API.
 */
export interface ISoftwareInstallResult {
  host_display_name?: string;
  install_uuid: string;
  software_title: string;
  software_title_id: number;
  software_package: string;
  host_id: number;
  status: SoftwareInstallStatus;
  detail: string;
  output: string;
  pre_install_query_output: string;
  post_install_script_output: string;
  created_at: string;
  updated_at: string | null;
  self_service: boolean;
}

export interface ISoftwareInstallResults {
  results: ISoftwareInstallResult;
}

// ISoftwareInstallerType defines the supported installer types for
// software uploaded by the IT admin.
export type ISoftwareInstallerType = "pkg" | "msi" | "deb" | "rpm" | "exe";

export interface ISoftwareLastInstall {
  install_uuid: string;
  installed_at: string;
}

export interface IAppLastInstall {
  command_uuid: string;
  installed_at: string;
}

interface SignatureInformation {
  installed_path: string;
  team_identifier: string;
  hash_sha256: string | null;
}
export interface ISoftwareLastUninstall {
  script_execution_id: string;
  uninstalled_at: string;
}

export interface ISoftwareInstallVersion {
  version: string;
  bundle_identifier: string;
  last_opened_at: string | null;
  vulnerabilities: string[] | null;
  installed_paths: string[];
  signature_information?: SignatureInformation[];
}

export interface IHostSoftwarePackage {
  name: string;
  self_service: boolean;
  icon_url: string | null;
  version: string;
  last_install: ISoftwareLastInstall | null;
  last_uninstall: ISoftwareLastUninstall | null;
  categories?: SoftwareCategory[];
  automatic_install_policies?: ISoftwareInstallPolicy[] | null;
}

export interface IHostAppStoreApp {
  app_store_id: string;
  self_service: boolean;
  icon_url: string;
  version: string;
  last_install: IAppLastInstall | null;
  categories?: SoftwareCategory[];
  automatic_install_policies?: ISoftwareInstallPolicy[] | null;
}

export interface IHostSoftware {
  id: number;
  name: string;
  software_package: IHostSoftwarePackage | null;
  app_store_app: IHostAppStoreApp | null;
  source: SoftwareSource;
  bundle_identifier?: string;
  status: Exclude<SoftwareInstallStatus, "uninstalled"> | null;
  installed_versions: ISoftwareInstallVersion[] | null;
}

export type IHostSoftwareUiStatus =
  | "installed"
  | "uninstalled"
  | "installing"
  | "uninstalling"
  | "updating"
  | "pending_install"
  | "pending_uninstall"
  | "pending_update"
  | "failed_install"
  | "failed_install_update_available"
  | "failed_uninstall"
  | "failed_uninstall_update_available"
  | "update_available";

/**
 * Extends IHostSoftware with a computed `ui_status` field.
 *
 * The `ui_status` categorizes software installation state for the UI by
 * combining the `status`, `installed_versions` info, and other factors
 * like host online state (via getUiStatus helper function), enabling
 * more detailed and status labels needed for the status and actions columns.
 */
export interface IHostSoftwareWithUiStatus extends IHostSoftware {
  ui_status: IHostSoftwareUiStatus;
}

/**
 * Allows unified data model for rendering of host VPP software installs and uninstalls
 * Optional as pending may not have a commandUuid
 */
export type IVPPHostSoftware = IHostSoftware & {
  commandUuid?: string;
};

export type IHostSoftwareUninstall = IHostSoftwareWithUiStatus & {
  scriptExecutionId: string;
};

export type IDeviceSoftware = IHostSoftware;
export type IDeviceSoftwareWithUiStatus = IHostSoftwareWithUiStatus;

const INSTALL_STATUS_PREDICATES: Record<
  SoftwareInstallStatus | "pending",
  string
> = {
  pending: "pending",
  installed: "installed",
  uninstalled: "uninstalled",
  pending_install: "told Fleet to install",
  failed_install: "failed to install",
  pending_uninstall: "told Fleet to uninstall",
  failed_uninstall: "failed to uninstall",
} as const;

export const getInstallStatusPredicate = (status: string | undefined) => {
  if (!status) {
    return INSTALL_STATUS_PREDICATES.pending;
  }
  return (
    INSTALL_STATUS_PREDICATES[status.toLowerCase() as SoftwareInstallStatus] ||
    INSTALL_STATUS_PREDICATES.pending
  );
};

export const aggregateInstallStatusCounts = (
  packageStatuses: ISoftwarePackage["status"]
) => ({
  installed: packageStatuses.installed,
  pending: packageStatuses.pending_install + packageStatuses.pending_uninstall,
  failed: packageStatuses.failed_install + packageStatuses.failed_uninstall,
});

export const INSTALL_STATUS_ICONS: Record<
  SoftwareInstallStatus | "pending" | "failed",
  IconNames
> = {
  pending: "pending-outline",
  pending_install: "pending-outline",
  installed: "success-outline",
  uninstalled: "success-outline",
  failed: "error-outline",
  failed_install: "error-outline",
  pending_uninstall: "pending-outline",
  failed_uninstall: "error-outline",
} as const;

type IHostSoftwarePackageWithLastInstall = IHostSoftwarePackage & {
  last_install: ISoftwareLastInstall;
};

export const hasHostSoftwarePackageLastInstall = (
  software: IHostSoftware
): software is IHostSoftware & {
  software_package: IHostSoftwarePackageWithLastInstall;
} => {
  return !!software.software_package?.last_install;
};

type IHostAppWithLastInstall = IHostAppStoreApp & {
  last_install: IAppLastInstall;
};

export const hasHostSoftwareAppLastInstall = (
  software: IHostSoftware
): software is IHostSoftware & {
  app_store_app: IHostAppWithLastInstall;
} => {
  return !!software.app_store_app?.last_install;
};

export const isIpadOrIphoneSoftwareSource = (source: string) =>
  ["ios_apps", "ipados_apps"].includes(source);

export interface IFleetMaintainedApp {
  id: number;
  name: string;
  version: string;
  platform: FleetMaintainedAppPlatform;
  software_title_id?: number; // null unless the team already has the software added (as a Fleet-maintained app, App Store (app), or custom package)
}

export type FleetMaintainedAppPlatform = Extract<
  Platform,
  "darwin" | "windows"
>;

export interface ICombinedFMA {
  name: string;
  macos: Omit<IFleetMaintainedApp, "name"> | null;
  windows: Omit<IFleetMaintainedApp, "name"> | null;
}
export interface IFleetMaintainedAppDetails {
  id: number;
  name: string;
  version: string;
  platform: FleetMaintainedAppPlatform;
  pre_install_script: string;
  install_script: string;
  post_install_script: string;
  uninstall_script: string;
  url: string;
  slug: string;
  software_title_id?: number; // null unless the team already has the software added (as a Fleet-maintained app, App Store (app), or custom package)
  categories: SoftwareCategory[];
}
