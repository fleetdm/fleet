import {
  IAppStoreApp,
  ISoftwareTitleDetails,
  isSoftwarePackage,
  aggregateInstallStatusCounts,
  SCRIPT_PACKAGE_SOURCES,
  ISoftwarePackage,
} from "interfaces/software";

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

// eslint-disable-next-line import/prefer-default-export
export const getInstallerCardInfo = (
  softwareTitle: ISoftwareTitleDetails
): InstallerCardInfo => {
  const installerData = softwareTitle.software_package
    ? softwareTitle.software_package
    : (softwareTitle.app_store_app as IAppStoreApp);

  const isPackage = isSoftwarePackage(installerData);

  return {
    softwareTitleName: softwareTitle.name,
    softwareDisplayName: softwareTitle.display_name || softwareTitle.name,
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
    autoUpdateStartTime: softwareTitle.auto_update_start_time,
    autoUpdateEndTime: softwareTitle.auto_update_end_time,
  };
};
