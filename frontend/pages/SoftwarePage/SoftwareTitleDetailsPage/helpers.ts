import {
  IAppStoreApp,
  ISoftwareTitleDetails,
  isSoftwarePackage,
  aggregateInstallStatusCounts,
} from "interfaces/software";

/**
 * Generates the data needed to render the installer card. It differentiates between
 * software packages and app store apps and returns the appropriate data.
 *
 * FIXME: This function ought to be refactored or renamed to better reflect its purpose.
 * "PackageCard" is a bit ambiguous in this context (it refers to the card that displays
 * package or app information, as applicable).
 */
// eslint-disable-next-line import/prefer-default-export
export const getInstallerCardInfo = (softwareTitle: ISoftwareTitleDetails) => {
  // we know at this point that softwareTitle.software_package or
  // softwareTitle.app_store_app is not null so we will do a type assertion.
  const installerData = softwareTitle.software_package
    ? softwareTitle.software_package
    : (softwareTitle.app_store_app as IAppStoreApp);

  const isPackage = isSoftwarePackage(installerData);

  return {
    softwarePackage: installerData,
    name: (isPackage && installerData.name) || softwareTitle.name,
    version:
      (isPackage ? installerData.version : installerData.latest_version) ||
      null,
    addedTimestamp: isPackage
      ? installerData.uploaded_at
      : installerData.created_at,
    status: isPackage
      ? aggregateInstallStatusCounts(installerData.status)
      : installerData.status,
    isSelfService: installerData.self_service,
  };
};
