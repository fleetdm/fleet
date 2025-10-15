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
  application_id?: string | null; // e.g., "us.zoom.videomeetings" for Android apps
  source: string; // "apps" | "ipados_apps" | "ios_apps" | "programs" | "rpm_packages" | "deb_packages" | "android_apps" | ?
  generated_cpe: string;
  vulnerabilities: ISoftwareVulnerability[] | null;
  hosts_count?: number;
  last_opened_at?: string | null; // e.g., "2021-08-18T15:11:35Z”
  installed_paths?: string[];
  extension_for?: string;
  vendor?: string;
  icon_url: string | null; // Only available on team view if an admin uploaded an icon to a team's software
}

export type IVulnerabilitySoftware = Omit<
  ISoftware,
  "vulnerabilities" | "icon_url"
> & {
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
  app_store_id: string; // API returns this as a string
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
  icon_url: string | null;
  versions_count: number;
  source: SoftwareSource;
  extension_for?: SoftwareExtensionFor;
  hosts_count: number;
  versions: ISoftwareTitleVersion[] | null;
  software_package: ISoftwarePackage | null;
  app_store_app: IAppStoreApp | null;
  /** @deprecated Use extension_for instead */
  browser?: string;
}

export interface ISoftwareTitleDetails {
  id: number;
  name: string;
  icon_url: string | null;
  software_package: ISoftwarePackage | null;
  app_store_app: IAppStoreApp | null;
  source: SoftwareSource;
  extension_for?: SoftwareExtensionFor;
  hosts_count: number;
  versions: ISoftwareTitleVersion[] | null;
  counts_updated_at?: string;
  bundle_identifier?: string;
  versions_count?: number;
  /** @deprecated Use extension_for instead */
  browser?: string;
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
  extension_for: SoftwareExtensionFor;
  release: string; // TODO: on software/verions/:id?
  vendor: string;
  arch: string; // e.g., "x86_64" // TODO: on software/verions/:id?
  generated_cpe: string;
  vulnerabilities: ISoftwareVulnerability[] | null;
  hosts_count?: number;
  /** @deprecated Use extension_for instead */
  browser?: string;
}

export const SOURCE_TYPE_CONVERSION = {
  apt_sources: "Package (APT)",
  deb_packages: "Package (deb)",
  portage_packages: "Package (Portage)",
  rpm_packages: "Package (RPM)",
  yum_sources: "Package (YUM)",
  pacman_packages: "Package (pacman)",
  npm_packages: "Package (NPM)",
  atom_packages: "Package (Atom)", // Atom packages were removed from software inventory. Mapping is maintained for backwards compatibility. (2023-12-04)
  python_packages: "Package (Python)",
  tgz_packages: "Package (tar)",
  apps: "Application (macOS)",
  ios_apps: "Application (iOS)",
  ipados_apps: "Application (iPadOS)",
  android_apps: "Application (Android)",
  chrome_extensions: "Browser plugin", // chrome_extensions can include any chrome-based browser (e.g., edge), so we rely instead on the `extension_for` field computed by Fleet server and fallback to this value if it is not present.
  firefox_addons: "Browser plugin (Firefox)",
  safari_extensions: "Browser plugin (Safari)",
  homebrew_packages: "Package (Homebrew)",
  programs: "Program (Windows)",
  ie_extensions: "Browser plugin (IE)",
  chocolatey_packages: "Package (Chocolatey)",
  pkg_packages: "Package (pkg)",
  vscode_extensions: "IDE extension", // vscode_extensions can include any vscode-based editor (e.g., Cursor, Trae, Windsurf), so we rely instead on the `extension_for` field computed by Fleet server and fallback to this value if it is not present.
  sh_packages: "Payload-free (Linux)",
  ps1_packages: "Payload-free (Windows)",
  jetbrains_plugins: "IDE extension", // jetbrains_plugins can include any JetBrains IDE (e.g., IntelliJ, PyCharm, WebStorm), so we rely instead on the `extension_for` field computed by Fleet server and fallback to this value if it is not present.
} as const;

export type SoftwareSource = keyof typeof SOURCE_TYPE_CONVERSION;

/** Map installable software source to platform  */
export const INSTALLABLE_SOURCE_PLATFORM_CONVERSION = {
  apt_sources: "linux",
  deb_packages: "linux",
  portage_packages: "linux",
  rpm_packages: "linux",
  yum_sources: "linux",
  pacman_packages: "linux",
  tgz_packages: "linux",
  npm_packages: null,
  atom_packages: null,
  python_packages: null,
  apps: "darwin",
  ios_apps: "ios",
  ipados_apps: "ipados",
  android_apps: "android", // 4.76 Currently hidden upstream as not installable
  chrome_extensions: null,
  firefox_addons: null,
  safari_extensions: null,
  homebrew_packages: "darwin",
  programs: "windows",
  ie_extensions: null,
  chocolatey_packages: "windows",
  pkg_packages: "darwin",
  vscode_extensions: null,
  sh_packages: "linux", // 4.76 Added support for Linux hosts only
  ps1_packages: "windows",
  jetbrains_plugins: null,
} as const;

export const SCRIPT_PACKAGE_SOURCES = ["sh_packages", "ps1_packages"];

export const NO_VERSION_OR_HOST_DATA_SOURCES = [
  "tgz_packages",
  ...SCRIPT_PACKAGE_SOURCES,
];

export type InstallableSoftwareSource = keyof typeof INSTALLABLE_SOURCE_PLATFORM_CONVERSION;

const EXTENSION_FOR_TYPE_CONVERSION = {
  // chrome versions
  chrome: "Chrome",
  chromium: "Chromium",
  opera: "Opera",
  yandex: "Yandex",
  brave: "Brave",
  edge: "Edge",
  edge_beta: "Edge Beta",

  // vscode versions
  vscode: "VSCode",
  vscode_insiders: "VSCode Insiders",
  vscodium: "VSCodium",
  vscodium_insiders: "VSCodium Insiders",
  trae: "Trae",
  windsurf: "Windsurf",
  cursor: "Cursor",

  // jebtbrains versions
  clion: "CLion",
  datagrip: "DataGrip",
  goland: "GoLand",
  intellij_idea: "IntelliJ IDEA",
  intellij_idea_community_edition: "IntelliJ IDEA Community Edition",
  phpstorm: "PhpStorm",
  pycharm: "PyCharm",
  pycharm_community_edition: "PyCharm Community Edition",
  resharper: "ReSharper",
  rider: "Rider",
  rubymine: "RubyMine",
  rust_rov: "RustRover",
  webstorm: "WebStorm",
} as const;

export type SoftwareExtensionFor =
  | keyof typeof EXTENSION_FOR_TYPE_CONVERSION
  | "";

export const formatSoftwareType = ({
  source,
  extension_for,
}: {
  source: SoftwareSource;
  extension_for?: SoftwareExtensionFor;
}) => {
  let type: string = SOURCE_TYPE_CONVERSION[source] || "Unknown";
  if (extension_for) {
    type += ` (${
      EXTENSION_FOR_TYPE_CONVERSION[extension_for] || startCase(extension_for)
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
] as const;

// Payload-free (script) software statuses
export const SOFTWARE_SCRIPT_STATUSES = [
  "ran_script",
  "pending_script",
  "failed_script",
] as const;

export type SoftwareInstallStatus = typeof SOFTWARE_INSTALL_STATUSES[number];

export const SOFTWARE_INSTALL_UNINSTALL_STATUSES = [
  ...SOFTWARE_INSTALL_STATUSES,
  ...SOFTWARE_UNINSTALL_STATUSES,
  // Payload-free (script) software statuses use API's SOFTWARE_INSTALL_STATUSES
] as const;

/*
 * SoftwareInstallUninstallStatus represents the possible states of software install operations.
 */
export type SoftwareInstallUninstallStatus = typeof SOFTWARE_INSTALL_UNINSTALL_STATUSES[number];

/** Include payload-free statuses */
export const ENAHNCED_SOFTWARE_INSTALL_UNINSTALL_STATUSES = [
  ...SOFTWARE_INSTALL_STATUSES,
  ...SOFTWARE_UNINSTALL_STATUSES,
  ...SOFTWARE_SCRIPT_STATUSES, // Payload-free (script) software
] as const;

/*
 * EnhancedSoftwareInstallUninstallStatus represents the possible states of software install operations including payload-free used in the UI.
 */
export type EnhancedSoftwareInstallUninstallStatus = typeof ENAHNCED_SOFTWARE_INSTALL_UNINSTALL_STATUSES[number];

export const isValidSoftwareInstallUninstallStatus = (
  s: string | undefined | null
): s is EnhancedSoftwareInstallUninstallStatus =>
  !!s &&
  ENAHNCED_SOFTWARE_INSTALL_UNINSTALL_STATUSES.includes(
    s as EnhancedSoftwareInstallUninstallStatus
  );

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
  status: SoftwareInstallUninstallStatus;
  detail: string;
  output: string;
  pre_install_query_output: string;
  post_install_script_output: string;
  created_at: string;
  updated_at: string | null;
  self_service: boolean;
}

// Script results are only install results, never uninstall
export type ISoftwareScriptResult = Omit<ISoftwareInstallResult, "status"> & {
  status: SoftwareInstallStatus;
};

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
  icon_url: string | null;
  software_package: IHostSoftwarePackage | null;
  app_store_app: IHostAppStoreApp | null;
  source: SoftwareSource;
  extension_for?: SoftwareExtensionFor;
  bundle_identifier?: string;
  status: Exclude<SoftwareInstallUninstallStatus, "uninstalled"> | null;
  installed_versions: ISoftwareInstallVersion[] | null;
}

/**
 * Comprehensive list of possible UI software statuses for host > software > library/self-service.
 *
 * These are more detailed than the raw API `.status` and are determined by:
 * - Whether the host is online or offline
 * - If the fleet-installed version is newer than any in installed_versions
 * - Special handling for tarballs (tgz_packages)
 * - Cases where the software inventory has not yet updated to reflect a recent change
 *   (i.e., last_install date vs host software's updated_at date)
 */
export type IHostSoftwareUiStatus =
  | "installed" // Present in inventory; no newer fleet installer version (tarballs: successful install only)
  | "uninstalled" // Not present in inventory (tarballs: successful uninstall or never installed)
  | "installing" // ONLINE; fleet-initiated install in progress
  | "uninstalling" // ONLINE; fleet-initiated uninstall in progress
  | "recently_updated" // Update applied (installer newer than inventory), but inventory not yet refreshed
  | "recently_installed" // Install applied (installer NOT newer than inventory), but inventory not yet refreshed
  | "recently_uninstalled" // Uninstall applied, but inventory not yet refreshed
  | "updating" // ONLINE; update (install) in progress with newer fleet installer
  | "pending_install" // OFFLINE; install scheduled (no newer installer version)
  | "pending_uninstall" // OFFLINE; uninstall scheduled
  | "pending_update" // OFFLINE; update scheduled (no newer installer version)
  | "failed_install" // Install attempt failed
  | "failed_install_update_available" // Install/update failed; newer installer version available
  | "failed_uninstall" // Uninstall attempt failed
  | "failed_uninstall_update_available" // Uninstall/update failed; newer installer version available
  | "update_available" // In inventory, but newer fleet installer version is available
  // Script UI statuses
  | "ran_script" // Script package ran successfully
  | "failed_script" // Script package failed to run
  | "running_script" // ONLINE; fleet-initiated script run in progress
  | "pending_script" // OFFLINE; fleet-initiated script run scheduled
  | "never_ran_script"; // Script package never ran before

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
  EnhancedSoftwareInstallUninstallStatus | "pending",
  string
> = {
  pending: "pending",
  installed: "installed",
  uninstalled: "uninstalled",
  pending_install: "told Fleet to install",
  failed_install: "failed to install",
  pending_uninstall: "told Fleet to uninstall",
  failed_uninstall: "failed to uninstall",
  ran_script: "ran", // Payload-free (script) software
  failed_script: "failed to run", // Payload-free (script) software
  pending_script: "told Fleet to run", // Payload-free (script) software
} as const;

export const getInstallUninstallStatusPredicate = (
  status: string | undefined,
  isScriptPackage = false
) => {
  if (!status) {
    return INSTALL_STATUS_PREDICATES.pending;
  }

  // If it is a script package, map install statuses to script-specific predicates
  if (isScriptPackage) {
    switch (status.toLowerCase()) {
      case "installed":
        return INSTALL_STATUS_PREDICATES.ran_script;
      case "pending_install":
        return INSTALL_STATUS_PREDICATES.pending_script;
      case "failed_install":
        return INSTALL_STATUS_PREDICATES.failed_script;
      default:
        break;
    }
  }

  // For all other cases, return the matching predicate or default to pending
  return (
    INSTALL_STATUS_PREDICATES[
      status.toLowerCase() as keyof typeof INSTALL_STATUS_PREDICATES
    ] || INSTALL_STATUS_PREDICATES.pending
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
  EnhancedSoftwareInstallUninstallStatus | "pending" | "failed",
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
  ran_script: "success-outline", // Payload-free (script) software
  failed_script: "error-outline", // Payload-free (script) software
  pending_script: "pending-outline", // Payload-free (script) software
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

export const isAndroidSoftwareSource = (source: string) =>
  source === "android_apps";

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
