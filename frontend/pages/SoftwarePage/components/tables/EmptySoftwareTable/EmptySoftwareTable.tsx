import React from "react";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import { IEmptyStateProps } from "interfaces/empty_state";
import {
  getVulnFilterRenderDetails,
  ISoftwareVulnFiltersParams,
} from "pages/SoftwarePage/SoftwareInventory/SoftwareInventoryTable/helpers";
import { ISoftwareDropdownFilterVal } from "pages/SoftwarePage/SoftwareLibrary/SoftwareLibraryTable/helpers";
import { HostPlatform, isAndroid } from "interfaces/platform";

export interface IEmptySoftwareTableProps {
  softwareFilter?: ISoftwareDropdownFilterVal;
  vulnFilters?: ISoftwareVulnFiltersParams;
  tableName?: string;
  isSoftwareDisabled?: boolean;
  noSearchQuery?: boolean;
  installableSoftwareExists?: boolean;
  platform?: HostPlatform;
}

const generateTypeText = (
  tableName: string,
  softwareFilter?: ISoftwareDropdownFilterVal,
  vulnFilters?: ISoftwareVulnFiltersParams
) => {
  if (softwareFilter === "installableSoftware") {
    return "installable software";
  }
  if (vulnFilters?.vulnerable) {
    return "vulnerable software";
  }
  return tableName;
};

const EmptySoftwareTable = ({
  softwareFilter = "allSoftware",
  vulnFilters,
  tableName = "software",
  isSoftwareDisabled,
  noSearchQuery,
  installableSoftwareExists,
  platform,
}: IEmptySoftwareTableProps): JSX.Element => {
  const softwareTypeText = generateTypeText(
    tableName,
    softwareFilter,
    vulnFilters
  );

  const { filterCount: vulnFiltersCount } = getVulnFilterRenderDetails(
    vulnFilters
  );

  const isFiltered =
    vulnFiltersCount > 0 || !noSearchQuery || softwareFilter !== "allSoftware";

  const getEmptySoftwareInfo = (): IEmptyStateProps => {
    if (isSoftwareDisabled) {
      return {
        header: "Software inventory disabled",
        info: (
          <>
            Users with the admin role can{" "}
            <CustomLink
              url="https://fleetdm.com/docs/using-fleet/vulnerability-processing#configuration"
              text="turn on software inventory"
              newTab
            />
            .
          </>
        ),
      };
    }

    let info = `Expecting to see ${softwareTypeText}? Check back later.`;
    if (isAndroid(platform || "")) {
      info = `${info} It may take up to 24 hours for Android to report the software.`;
    }

    if (!isFiltered) {
      if (softwareFilter === "allSoftware") {
        if (installableSoftwareExists) {
          return {
            header: `No ${tableName} detected`,
            info: "Install software on your hosts to see versions.",
          };
        }
        return {
          header: `No ${tableName} detected`,
          info,
        };
      }
    }

    return {
      header: "No items match the current search criteria",
      info,
    };
  };

  const emptySoftware = getEmptySoftwareInfo();

  return <EmptyState header={emptySoftware.header} info={emptySoftware.info} />;
};

export default EmptySoftwareTable;
