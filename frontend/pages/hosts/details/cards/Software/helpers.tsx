import { QueryParams } from "utilities/url";
import { Row } from "react-table";
import { flatMap } from "lodash";
import { HostPlatform, isIPadOrIPhone } from "interfaces/platform";
import { MdmEnrollmentStatus } from "interfaces/mdm";
import {
  IHostSoftware,
  IHostSoftwareUiStatus,
  IHostSoftwareWithUiStatus,
} from "interfaces/software";
import { IconNames } from "components/icons";
import { getLastInstall } from "../HostSoftwareLibrary/helpers";

// available_for_install string > boolean conversion in parseHostSoftwareQueryParams
export const getHostSoftwareFilterFromQueryParams = (
  queryParams: QueryParams
) => {
  const { available_for_install } = queryParams;

  return available_for_install ? "installableSoftware" : "allSoftware";
};

// VERSION COMPARISON UTILITIES FOR SOFTWARE VERSIONS

// Order of pre-release tags for version comparison
const PRE_RELEASE_ORDER = ["alpha", "beta", "rc", ""];

/**
 * Removes build metadata from a version string (e.g., "1.0.0+build" -> "1.0.0").
 */
const stripBuildMetadata = (version: string): string => version.split("+")[0];

/**
 * Splits a version string into an array of numeric and string segments.
 * Handles delimiters, pre-release tags, and normalizes case.
 */
const splitVersion = (version: string): Array<string | number> => {
  if (typeof version !== "string" || !version.trim()) {
    // Defensive: handle null, undefined, or empty version strings
    console.warn(
      `Warning: Invalid version used for version comparison: "${version}"`
    );
    return [0]; // fallback to [0] as a safe default
  }
  // Normalize delimiters and strip build metadata
  return flatMap(
    stripBuildMetadata(version).replace(/[-_]/g, ".").split("."),
    (part: string) => part.match(/\d+|[a-zA-Z]+/g) || []
  ).map((seg: string) => {
    if (/^\d+$/.test(seg)) {
      // numeric segment, convert to number
      return Number(seg);
    } else if (/^[a-zA-Z]+$/.test(seg)) {
      // expected alphabetic segment, normalize to lowercase
      return seg.toLowerCase();
    }
    // unexpected, possibly malformed
    console.warn(
      `Warning: Unexpected version segment "${seg}" found in version string "${version}"`
    );
    // fallback: return as lowercase string anyway
    return seg.toLowerCase();
  });
};

/**
 * Compares two pre-release identifiers according to PRE_RELEASE_ORDER.
 * Returns -1 if a < b, 1 if a > b, 0 if equal.
 */
const comparePreRelease = (a: string, b: string): number => {
  const idxA = PRE_RELEASE_ORDER.indexOf(a);
  const idxB = PRE_RELEASE_ORDER.indexOf(b);
  if (idxA === -1 && idxB === -1) return a.localeCompare(b);
  if (idxA === -1) return 1;
  if (idxB === -1) return -1;
  if (idxA < idxB) return -1;
  if (idxA > idxB) return 1;
  return 0;
};

/**
 * Compares two software version strings.
 * Returns:
 *   -1 if v1 < v2
 *    0 if v1 === v2
 *    1 if v1 > v2
 * Handles semantic versioning, pre-release tags, and build metadata.
 * See helpers.tests.ts for examples and edge cases.
 *
 * Note: This is more robust than /utilities/helpers.tsx compareVersions function
 * which only splits on . and is not suitable for prerelese, metadata, and non-trad schemes
 *
 * Pitfalls & Known Limitations:
 * - Designed primarily for Semantic Versioning (SemVer, e.g., "1.2.3-alpha").
 * - May produce unexpected results with **non-SemVer** versions:
 *   - Versions using formats with build metadata or unconventional separators (e.g., "1.2.3+build.1" or "2023-07-15").
 *   - Pre-release and metadata handling depends on correct and complete `PRE_RELEASE_ORDER` and `comparePreRelease` helper.
 * - Versions with **mixed types per segment** (e.g., "1.0.0-beta" vs "1.0.0-5") may not follow SemVer rules precisely.
 */
export const compareVersions = (v1: string, v2: string): number => {
  if (typeof v1 !== "string" || typeof v2 !== "string") {
    console.warn(
      "Warning: Version comparison received non-string input.",
      v1,
      v2
    );
    return 0;
  }

  const s1 = splitVersion(v1);
  const s2 = splitVersion(v2);
  const maxLen = Math.max(s1.length, s2.length);
  let result = 0;

  // Compare each version segment one by one, left-to-right, until a difference is found
  // Stop on the first difference between two versions
  Array.from({ length: maxLen }).some((_, i) => {
    const a = s1[i] ?? 0;
    const b = s2[i] ?? 0;

    if (typeof a === "number" && typeof b === "number") {
      if (a !== b) {
        result = a > b ? 1 : -1;
        return true;
      }
    } else if (typeof a === "string" && typeof b === "string") {
      // Compare pre-release tags if present
      if (PRE_RELEASE_ORDER.includes(a) || PRE_RELEASE_ORDER.includes(b)) {
        const cmp = comparePreRelease(a, b);
        if (cmp !== 0) {
          result = cmp;
          return true;
        }
      } else if (a !== b) {
        result = a > b ? 1 : -1;
        return true;
      }
    } else {
      // When comparing segments at the same position, numbers > strings (e.g., 1.0 > 1.0-beta, 1.2.3 > 1.2-rc)
      result = typeof a === "number" ? 1 : -1;
      return true;
    }
    return false;
  });

  return result;
};

// INSTALLER UTILITIES

const getInstallerVersion = (software: IHostSoftware) => {
  if (software.software_package && software.software_package.version) {
    return software.software_package.version;
  }
  if (software.app_store_app && software.app_store_app.version) {
    return software.app_store_app.version;
  }
  return null;
};

// UI_STATUS UTILITIES

const getNewerDate = (dateStr1: string, dateStr2: string) => {
  return dateStr1 > dateStr2 ? dateStr1 : dateStr2;
};

export const getUiStatus = (
  software: IHostSoftware,
  isHostOnline: boolean,
  hostSoftwareUpdatedAt?: string | null
): IHostSoftwareUiStatus => {
  const { status, installed_versions } = software;

  const lastInstallDate = getLastInstall(software)?.installed_at;
  const installerVersion = getInstallerVersion(software);

  // If the installation has failed, return 'failed_install'
  if (status === "failed_install") {
    if (
      installerVersion &&
      installed_versions &&
      installed_versions.some(
        (iv) => compareVersions(iv.version, installerVersion) === -1
      )
    ) {
      return "failed_install_update_available";
    }
    return "failed_install";
  }

  // If the uninstallation has failed, return 'failed_uninstall'
  if (status === "failed_uninstall") {
    if (
      installerVersion &&
      installed_versions &&
      installed_versions.some(
        (iv) => compareVersions(iv.version, installerVersion) === -1
      )
    ) {
      return "failed_uninstall_update_available";
    }
    return "failed_uninstall";
  }

  // If installation is pending
  if (status === "pending_install") {
    if (
      installed_versions &&
      installed_versions.length > 0 &&
      installerVersion
    ) {
      // Are we updating (installerVersion > installed), or reinstalling (installerVersion == installed)?
      const isUpdate = installed_versions.some(
        (iv) => compareVersions(iv.version, installerVersion) === -1
      );

      // Updating to a newer version
      if (isUpdate) {
        return isHostOnline ? "updating" : "pending_update";
      }
    }
    // Reinstalling equivalent versions or installing with no currently installed versions
    return isHostOnline ? "installing" : "pending_install";
  }

  // If uninstallation is pending
  if (status === "pending_uninstall") {
    // Return 'uninstalling' if host is online, else 'pending_uninstall'
    return isHostOnline ? "uninstalling" : "pending_uninstall";
  }

  // Check if any installed version is less than the installer version, indicating an update is available
  if (
    installerVersion &&
    installed_versions &&
    installed_versions.some(
      (iv) => compareVersions(iv.version, installerVersion) === -1
    )
  ) {
    const newerDate =
      hostSoftwareUpdatedAt && lastInstallDate
        ? getNewerDate(hostSoftwareUpdatedAt, lastInstallDate)
        : lastInstallDate;

    return newerDate === lastInstallDate ? "updated" : "update_available";
  }

  // Tgz packages that are installed via Fleet should return 'installed' as they
  // are not tracked in software inventory (installed_versions)
  if (software.source === "tgz_packages" && software.status === "installed") {
    return "installed";
  }

  // If there are installed versions and none need updating, return 'installed'
  if (installed_versions && installed_versions.length > 0) return "installed";

  // Default fallback status when no other conditions are met
  return "uninstalled"; // fallback
};

// Library/Self-Service Action Button Configurations
export interface IButtonDisplayConfig {
  install: {
    text: string;
    icon: IconNames;
  };
  uninstall: {
    text: string;
    icon: IconNames;
  };
}

type ButtonType = "install" | "uninstall";

interface IButtonConfig {
  text: string;
  icon: IconNames;
}

/** Display text and icon are shared across self-service and
 * host details > library action buttons */
export const getInstallerActionButtonConfig = (
  type: ButtonType,
  status: IHostSoftwareUiStatus
): IButtonConfig => {
  if (type === "install") {
    switch (status) {
      case "failed_install":
      case "failed_install_update_available":
        return { text: "Retry", icon: "refresh" };
      case "installed":
      case "pending_uninstall":
      case "uninstalling":
      case "failed_uninstall":
        return { text: "Reinstall", icon: "refresh" };
      case "pending_update":
      case "updating":
      case "update_available":
      case "failed_uninstall_update_available":
        return { text: "Update", icon: "refresh" };
      default:
        return { text: "Install", icon: "install" };
    }
  } else {
    // uninstall
    switch (status) {
      case "failed_uninstall":
      case "failed_uninstall_update_available":
        return { text: "Retry uninstall", icon: "refresh" };
      default:
        return { text: "Uninstall", icon: "trash" };
    }
  }
};

// Install status sorting utilities

const INSTALL_STATUS_SORT_ORDER: IHostSoftwareUiStatus[] = [
  "failed_install", // Failed
  "failed_install_update_available", // Failed install with update available
  "failed_uninstall", // Failed uninstall
  "failed_uninstall_update_available", // Failed uninstall with update available
  "update_available", // Update available
  "updating", // Updating...
  "pending_update", // Update (pending)
  "installing", // Installing...
  "pending_install", // Install (pending)
  "uninstalling", // Uninstalling...
  "pending_uninstall", // Uninstall (pending)
  "installed", // Installed
  "uninstalled", // Empty (---)
];

/** Status column custom sortType */
export const installStatusSortType = (
  rowA: Row<IHostSoftwareWithUiStatus>,
  rowB: Row<IHostSoftwareWithUiStatus>,
  columnId: string
) => {
  // Type assertion ensures only valid status strings or undefined
  const statusA = rowA.original[columnId as keyof IHostSoftwareWithUiStatus] as
    | IHostSoftwareUiStatus
    | undefined;
  const statusB = rowB.original[columnId as keyof IHostSoftwareWithUiStatus] as
    | IHostSoftwareUiStatus
    | undefined;

  const indexA = INSTALL_STATUS_SORT_ORDER.indexOf(statusA ?? "uninstalled");
  const indexB = INSTALL_STATUS_SORT_ORDER.indexOf(statusB ?? "uninstalled");

  // If not found, put at end
  const safeIndexA = indexA === -1 ? INSTALL_STATUS_SORT_ORDER.length : indexA;
  const safeIndexB = indexB === -1 ? INSTALL_STATUS_SORT_ORDER.length : indexB;

  if (safeIndexA < safeIndexB) return -1;
  if (safeIndexA > safeIndexB) return 1;
  return 0;
};

interface IGetSoftwareSubheader {
  platform: HostPlatform;
  hostMdmEnrollmentStatus: MdmEnrollmentStatus | null;
  isMyDevicePage?: boolean;
}

/**
 * Returns a subheader string for the software page based on platform and MDM enrollment status.
 * Handles iOS-specific cases for personal and manual MDM enrollment.
 */
export const getSoftwareSubheader = ({
  platform,
  hostMdmEnrollmentStatus,
  isMyDevicePage,
}: IGetSoftwareSubheader): string => {
  if (isIPadOrIPhone(platform)) {
    if (hostMdmEnrollmentStatus === "On (personal)") {
      return isMyDevicePage
        ? "Software installed on your work profile (Managed Apple Account)."
        : "Software installed on work profile (Managed Apple Account).";
    }
    if (hostMdmEnrollmentStatus === "On (manual)") {
      return isMyDevicePage
        ? "Software installed on your device. Built-in apps (e.g. Calculator) aren't included."
        : "Software installed on this host. Built-in apps (e.g. Calculator) aren't included.";
    }
  }
  return isMyDevicePage
    ? "Software installed on your device."
    : "Software installed on this host.";
};
