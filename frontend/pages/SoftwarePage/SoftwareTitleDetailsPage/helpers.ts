import {
  IAppStoreApp,
  ISoftwareTitleDetails,
  isSoftwarePackage,
  aggregateInstallStatusCounts,
  SCRIPT_PACKAGE_SOURCES,
  ISoftwarePackage,
  IFleetMaintainedVersion,
} from "interfaces/software";
import endpoints from "utilities/endpoints";
import URL_PREFIX from "router/url_prefix";
import { getDisplayedSoftwareName } from "../helpers";
import { deriveAccordionRowState } from "./LibraryItemAccordion/helpers";
import { LibraryItemBadgeState } from "./LibraryItemAccordion/LibraryItemAccordion";

/** Row-level gate for the LibraryItemAccordion download button: shown only
 * when the row is the active (currently-serving) version AND the user has
 * installer read permission. Observers and any role excluded from
 * `installable_entity` read in policy.rego return `false` here so the button
 * is hidden rather than clicked into a backend 403. */
export const canDownloadInstallerRow = (
  rowIsActive: boolean,
  hasInstallerReadPermission: boolean
): boolean => rowIsActive && hasInstallerReadPermission;

/** Resolves which package to download: the explicitly-clicked row on a
 * multi-package title, or the title's first-added `software_package` for
 * single-package callers. Returns null when neither is available (e.g.,
 * an app-store title with no `software_package`). */
export const resolveDownloadTarget = (
  clicked: ISoftwarePackage | undefined,
  fallback: ISoftwarePackage | null | undefined
): ISoftwarePackage | null => clicked ?? fallback ?? null;

/** Builds the unauthenticated token-based download URL that the browser hits
 * via a synthetic `<a download>` click. */
export const buildInstallerDownloadUrl = (
  softwareTitleId: number,
  token: string,
  origin: string = global.window.location.origin
): string =>
  `${origin}${URL_PREFIX}/api${endpoints.SOFTWARE_PACKAGE_TOKEN(
    softwareTitleId
  )}/${token}`;

export interface InstallerCardInfo {
  softwareTitleName: string;
  softwareDisplayName: string;
  softwareInstaller: ISoftwarePackage | IAppStoreApp;
  name: string;
  version: string | null;
  source: ISoftwareTitleDetails["source"];
  addedTimestamp: string;
  status: {
    installed: number;
    pending: number;
    failed: number;
  };
  isSelfService: boolean;
  isScriptPackage: boolean;
  iconUrl?: string | null;
  displayName?: string;
  autoUpdateEnabled?: boolean;
  autoUpdateStartTime?: string;
  autoUpdateEndTime?: string;
}

export interface ILibraryVersionRow {
  id: number;
  version: string;
  filename?: string;
  uploaded_at: string;
  isActive: boolean;
  badgeState?: LibraryItemBadgeState;
}

/** Builds the Library accordion rows for a title: one row per cached
 * Fleet-maintained version (the active row badged from the pin, the rest dimmed
 * rollback candidates), or a single active un-badged row for installer types
 * that have no cached-version list. The `latest`/`pinned`/`majorVersion` badges
 * are FMA-semantics — custom packages have no version history to pin against,
 * so no badge is rendered. */
export const buildLibraryVersionRows = ({
  fleetMaintainedVersions,
  activeVersion,
  pinnedVersion,
  addedTimestamp,
}: {
  fleetMaintainedVersions?: IFleetMaintainedVersion[] | null;
  activeVersion: string | null;
  pinnedVersion?: string | null;
  addedTimestamp: string;
}): ILibraryVersionRow[] => {
  if (fleetMaintainedVersions?.length) {
    return fleetMaintainedVersions.map((v) => ({
      ...v,
      ...deriveAccordionRowState({
        rowVersion: v.version,
        activeVersion,
        pinnedVersion,
      }),
    }));
  }
  return [
    {
      id: -1,
      version: activeVersion ?? "",
      uploaded_at: addedTimestamp,
      isActive: true,
    },
  ];
};

export const getInstallerCardInfo = (
  softwareTitle: ISoftwareTitleDetails
): InstallerCardInfo => {
  const installerData = softwareTitle.software_package
    ? softwareTitle.software_package
    : (softwareTitle.app_store_app as IAppStoreApp);

  const isPackage = isSoftwarePackage(installerData);

  return {
    softwareTitleName: softwareTitle.name,
    softwareDisplayName: getDisplayedSoftwareName(
      softwareTitle.name,
      softwareTitle.display_name
    ),
    softwareInstaller: installerData,
    name: (isPackage && installerData.name) || softwareTitle.name,
    version:
      (isPackage ? installerData.version : installerData.latest_version) ||
      null,
    source: softwareTitle.source,
    iconUrl: softwareTitle.icon_url,
    displayName: softwareTitle.display_name,
    addedTimestamp: isPackage
      ? installerData.uploaded_at
      : installerData.created_at,
    status: isPackage
      ? aggregateInstallStatusCounts(installerData.status)
      : installerData.status,
    isSelfService: installerData.self_service,
    isScriptPackage:
      SCRIPT_PACKAGE_SOURCES.includes(softwareTitle.source) || false,
    autoUpdateEnabled: softwareTitle.auto_update_enabled,
    autoUpdateStartTime: softwareTitle.auto_update_window_start,
    autoUpdateEndTime: softwareTitle.auto_update_window_end,
  };
};
