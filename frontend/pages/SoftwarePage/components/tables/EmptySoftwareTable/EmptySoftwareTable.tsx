import React from "react";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import { IEmptyStateProps } from "interfaces/empty_state";
import {
  getVulnFilterRenderDetails,
  ISoftwareVulnFiltersParams,
} from "pages/SoftwarePage/SoftwareInventory/SoftwareInventoryTable/helpers";
import { HostPlatform, isAndroid } from "interfaces/platform";

export interface IEmptySoftwareTableProps {
  vulnFilters?: ISoftwareVulnFiltersParams;
  tableName?: string;
  isSoftwareDisabled?: boolean;
  noSearchQuery?: boolean;
  installableSoftwareExists?: boolean;
  platform?: HostPlatform;
}

/** Maps tableName to the truly-empty (no filters) info text. */
const EMPTY_INFO_BY_TABLE: Record<string, string> = {
  software:
    "Recently installed software will appear after the next scheduled check-in.",
  "operating systems":
    "Operating system data will appear after the next scheduled check-in.",
  vulnerabilities:
    "Vulnerability data will appear after the next scheduled check-in.",
};

/** Returns the display name used in filtered info text (e.g. "vulnerable software"). */
const getFilteredTypeText = (
  tableName: string,
  vulnFilters?: ISoftwareVulnFiltersParams
): string => {
  if (vulnFilters?.vulnerable) {
    return "vulnerable software";
  }
  return tableName;
};

const EmptySoftwareTable = ({
  vulnFilters,
  tableName = "software",
  isSoftwareDisabled,
  noSearchQuery = true,
  installableSoftwareExists,
  platform,
}: IEmptySoftwareTableProps): JSX.Element => {
  const { filterCount: vulnFiltersCount } = getVulnFilterRenderDetails(
    vulnFilters
  );

  const isFiltered = vulnFiltersCount > 0 || !noSearchQuery;

  const getEmptyStateProps = (): IEmptyStateProps => {
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
      if (installableSoftwareExists) {
        return {
          header: `No ${tableName} detected`,
          info: "Install software on your hosts to see versions.",
        };
      }

      let info = EMPTY_INFO_BY_TABLE[tableName] ?? EMPTY_INFO_BY_TABLE.software;
      if (isAndroid(platform || "")) {
        info = `${info} It may take up to 24 hours for Android to report the software.`;
      }

      return {
        header: `No ${tableName} detected`,
        info,
      };
    }

    // Filtered/search state: use type-aware text (e.g. "vulnerable software")
    const typeText = getFilteredTypeText(tableName, vulnFilters);
    let info = `Expecting to see ${typeText}? Check back later.`;
    if (isAndroid(platform || "")) {
      info = `${info} It may take up to 24 hours for Android to report the software.`;
    }

    return {
      header: "No items match the current search criteria",
      info,
    };
  };

  const emptyState = getEmptyStateProps();

  return <EmptyState header={emptyState.header} info={emptyState.info} />;
};

export default EmptySoftwareTable;
