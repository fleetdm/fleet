import { startCase } from "lodash";
import PropTypes from "prop-types";

import { IconNames } from "components/icons";

import vulnerabilityInterface from "./vulnerability";

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
  source: string; // "apps" | "ipados" | "ios" | "programs" | ?
  generated_cpe: string;
  vulnerabilities: ISoftwareVulnerability[] | null;
  hosts_count?: number;
  last_opened_at?: string | null; // e.g., "2021-08-18T15:11:35Z”
  installed_paths?: string[];
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

export interface ISoftwarePackage {
  name: string;
  version: string;
  uploaded_at: string;
  install_script: string;
  pre_install_query?: string;
  post_install_script?: string;
  self_service: boolean;
  icon_url: string | null;
  status: {
    installed: number;
    pending: number;
    failed: number;
  };
}

export const isSoftwarePackage = (
  data: ISoftwarePackage | IAppStoreApp
): data is ISoftwarePackage =>
  (data as ISoftwarePackage).install_script !== undefined;

export interface IAppStoreApp {
  name: string;
  app_store_id: number;
  latest_version: string;
  icon_url: string;
  status: {
    installed: number;
    pending: number;
    failed: number;
  };
}

export interface ISoftwareTitle {
  id: number;
  name: string;
  versions_count: number;
  source: string; // ipados_apps | ios_apps | apps | ...
  hosts_count: number;
  versions: ISoftwareTitleVersion[] | null;
  software_package: ISoftwarePackage | null;
  app_store_app: IAppStoreApp | null;
  browser?: string;
}

export interface ISoftwareTitleDetails {
  id: number;
  name: string;
  software_package: ISoftwarePackage | null;
  app_store_app: IAppStoreApp | null;
  source: string; // "apps" | "ios_apps" | "ipados_apps" | ?
  hosts_count: number;
  versions: ISoftwareTitleVersion[] | null;
  bundle_identifier?: string;
  browser?: string;
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
  source: string; // "apps" | "ipados_apps" | "ios_apps" | ?
  browser: string; // e.g., "chrome"
  release: string; // TODO: on software/verions/:id?
  vendor: string;
  arch: string; // e.g., "x86_64" // TODO: on software/verions/:id?
  generated_cpe: string;
  vulnerabilities: ISoftwareVulnerability[] | null;
  hosts_count?: number;
}

export const SOURCE_TYPE_CONVERSION: Record<string, string> = {
  apt_sources: "Package (APT)",
  deb_packages: "Package (deb)",
  portage_packages: "Package (Portage)",
  rpm_packages: "Package (RPM)",
  yum_sources: "Package (YUM)",
  npm_packages: "Package (NPM)",
  atom_packages: "Package (Atom)", // Atom packages were removed from software inventory. Mapping is maintained for backwards compatibility. (2023-12-04)
  python_packages: "Package (Python)",
  apps: "Application (macOS)",
  ios: "Application (iOS)",
  ipados: "Application (iPadOS)",
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

const BROWSER_TYPE_CONVERSION: Record<string, string> = {
  chrome: "Chrome",
  chromium: "Chromium",
  opera: "Opera",
  yandex: "Yandex",
  brave: "Brave",
  edge: "Edge",
  edge_beta: "Edge Beta",
} as const;

export const formatSoftwareType = ({
  source,
  browser,
}: {
  source: string;
  browser?: string;
}) => {
  let type = SOURCE_TYPE_CONVERSION[source] || "Unknown";
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
export const SOFTWARE_INSTALL_STATUSES = [
  "failed",
  "installed",
  "pending",
] as const;

/*
 * SoftwareInstallStatus represents the possible states of software install operations.
 */
export type SoftwareInstallStatus = typeof SOFTWARE_INSTALL_STATUSES[number];

export const isValidSoftwareInstallStatus = (
  s: string | undefined
): s is SoftwareInstallStatus =>
  !!s && SOFTWARE_INSTALL_STATUSES.includes(s as SoftwareInstallStatus);

/**
 * ISoftwareInstallResult is the shape of a software install result object
 * returned by the Fleet API.
 */
export interface ISoftwareInstallResult {
  install_uuid: string;
  software_title: string;
  software_title_id: number;
  software_package: string;
  host_id: number;
  host_display_name: string;
  status: SoftwareInstallStatus;
  detail: string;
  output: string;
  pre_install_query_output: string;
  post_install_script_output: string;
}

export interface ISoftwareInstallResults {
  results: ISoftwareInstallResult;
}

// ISoftwareInstallerType defines the supported installer types for
// software uploaded by the IT admin.
export type ISoftwareInstallerType = "pkg" | "msi" | "deb" | "exe";

export interface ISoftwareLastInstall {
  install_uuid: string;
  installed_at: string;
}

export interface IAppLastInstall {
  command_uuid: string;
  installed_at: string;
}

export interface ISoftwareInstallVersion {
  version: string;
  last_opened_at: string | null;
  vulnerabilities: string[] | null;
  installed_paths: string[];
}

export interface IHostSoftwarePackage {
  name: string;
  self_service: boolean;
  icon_url: string;
  version: string;
  last_install: ISoftwareLastInstall | null;
}

export interface IHostAppStoreApp {
  app_store_id: string;
  self_service: boolean;
  icon_url: string;
  version: string;
  last_install: IAppLastInstall | null;
}

export interface IHostSoftware {
  id: number;
  name: string;
  software_package: IHostSoftwarePackage | null;
  app_store_app: IHostAppStoreApp | null;
  source: string;
  bundle_identifier?: string;
  status: SoftwareInstallStatus | null;
  installed_versions: ISoftwareInstallVersion[] | null;
}

export type IDeviceSoftware = IHostSoftware;

const INSTALL_STATUS_PREDICATES: Record<SoftwareInstallStatus, string> = {
  failed: "failed to install",
  installed: "installed",
  pending: "told Fleet to install",
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

export const INSTALL_STATUS_ICONS: Record<SoftwareInstallStatus, IconNames> = {
  pending: "pending-outline",
  installed: "success-outline",
  failed: "error-outline",
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
