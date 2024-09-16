import {
  IAppStoreApp,
  ISoftwarePackage,
  ISoftwareTitleDetails,
  isSoftwarePackage,
} from "interfaces/software";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

const mergePackageStatuses = (packageStatuses: ISoftwarePackage["status"]) => ({
  installed: packageStatuses.installed,
  pending: packageStatuses.pending_install + packageStatuses.pending_uninstall,
  failed: packageStatuses.failed_install + packageStatuses.failed_uninstall,
});
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
    softwarePackage: isPackage ? packageData : undefined,
    name: (isPackage && packageData.name) || softwareTitle.name,
    version:
      (isSoftwarePackage(packageData)
        ? packageData.version
        : packageData.latest_version) || DEFAULT_EMPTY_CELL_VALUE,
    uploadedAt: isSoftwarePackage(packageData) ? packageData.uploaded_at : "",
    status: isSoftwarePackage(packageData)
      ? mergePackageStatuses(packageData.status)
      : packageData.status,
    isSelfService: packageData.self_service,
  };
};
