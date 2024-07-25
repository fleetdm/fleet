import {
  IAppStoreApp,
  ISoftwareTitleDetails,
  isSoftwarePackage,
} from "interfaces/software";

/**
 * Generates the data needed to render the package card.
 */
// eslint-disable-next-line import/prefer-default-export
export const getPackageCardInfo = (softwareTitle: ISoftwareTitleDetails) => {
  // we know at this point that softwareTitle.software_package or
  // softwareTitle.app_store_app is not null so we will do a type assertion.
  const packageData = softwareTitle.software_package
    ? softwareTitle.software_package
    : (softwareTitle.app_store_app as IAppStoreApp);

  return {
    softwarePackage: isSoftwarePackage(packageData) ? packageData : undefined,
    name: softwareTitle.name,
    // TODO(jacob) confirm desired behavior for "version not present" with Noah
    version: isSoftwarePackage(packageData)
      ? packageData.version
      : packageData.latest_version,
    uploadedAt: isSoftwarePackage(packageData) ? packageData.uploaded_at : "",
    status: packageData.status,
    isSelfService: isSoftwarePackage(packageData)
      ? packageData.self_service
      : false,
  };
};
