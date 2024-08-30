import React from "react";
import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";
import { IEmptyTableProps } from "interfaces/empty_table";
import {
  getVulnFilterRenderDetails,
  ISoftwareDropdownFilterVal,
  ISoftwareVulnFiltersParams,
} from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/helpers";

export interface IEmptySoftwareTableProps {
  softwareFilter?: ISoftwareDropdownFilterVal;
  vulnFilters?: ISoftwareVulnFiltersParams;
  tableName?: string;
  isSoftwareDisabled?: boolean;
  noSearchQuery?: boolean;
  installableSoftwareExists?: boolean;
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

  const getEmptySoftwareInfo = (): IEmptyTableProps => {
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
          info: `Expecting to see ${softwareTypeText}? Check back later.`,
        };
      }
    }

    return {
      header: "No items match the current search criteria",
      info: `Expecting to see ${softwareTypeText}? Check back later.`,
    };
  };

  const emptySoftware = getEmptySoftwareInfo();

  return (
    <EmptyTable
      graphicName="empty-search-question"
      header={emptySoftware.header}
      info={emptySoftware.info}
    />
  );
};

export default EmptySoftwareTable;
