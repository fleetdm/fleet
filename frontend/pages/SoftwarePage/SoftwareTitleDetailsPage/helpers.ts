import {
  IAppStoreApp,
  ISoftwareTitleDetails,
  isSoftwarePackage,
  aggregateInstallStatusCounts,
} from "interfaces/software";

/**
 * Generates the data needed to render the package card. It differentiates between
 * software packages and app store apps and returns the appropriate data.
 *
 * FIXME: This function ought to be refactored or renamed to better reflect its purpose.
 * "PackageCard" is a bit ambiguous in this context (it refers to the card that displays
 * package or app information, as applicable).
 */
// eslint-disable-next-line import/prefer-default-export
export const getPackageCardInfo = (softwareTitle: ISoftwareTitleDetails) => {
  // we know at this point that softwareTitle.software_package or
  // softwareTitle.app_store_app is not null so we will do a type assertion.
  const packageData = softwareTitle.software_package
    ? softwareTitle.software_package
    : (softwareTitle.app_store_app as IAppStoreApp);

  const isPackage = isSoftwarePackage(packageData);

  return {
    softwarePackage: packageData,
    name: (isPackage && packageData.name) || softwareTitle.name,
    version:
      (isSoftwarePackage(packageData)
        ? packageData.version
        : packageData.latest_version) || null,
    uploadedAt: isSoftwarePackage(packageData) ? packageData.uploaded_at : "",
    status: isSoftwarePackage(packageData)
      ? aggregateInstallStatusCounts(packageData.status)
      : packageData.status,
    isSelfService: packageData.self_service,
  };
};
