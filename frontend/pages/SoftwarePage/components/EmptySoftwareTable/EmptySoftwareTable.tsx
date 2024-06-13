// This component is used on DashboardPage.tsx > Software.tsx,
// Host Details / Device User > Software.tsx, and SoftwarePage.tsx

import React from "react";

import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";
import { IEmptyTableProps } from "interfaces/empty_table";
import { ISoftwareDropdownFilterVal } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/helpers";

export interface IEmptySoftwareTableProps {
  softwareFilter?: ISoftwareDropdownFilterVal;
  tableName?: string;
  isSoftwareDisabled?: boolean;
  isCollectingSoftware?: boolean;
  isSearching?: boolean;
}

const generateTypeText = (
  tableName: string,
  softwareFilter?: ISoftwareDropdownFilterVal
) => {
  if (softwareFilter === "installableSoftware") {
    return "installable software";
  }
  if (softwareFilter === "vulnerableSoftware") {
    return "vulnerable software";
  }
  return tableName;
};

const EmptySoftwareTable = ({
  softwareFilter,
  tableName = "software",
  isSoftwareDisabled,
  isCollectingSoftware,
  isSearching,
}: IEmptySoftwareTableProps): JSX.Element => {
  const softwareTypeText = generateTypeText(tableName, softwareFilter);

  const emptySoftware: IEmptyTableProps = {
    header: "No items match the current search criteria",
    info: `Expecting to see ${softwareTypeText}? Check back later.`,
  };

  if (isCollectingSoftware) {
    emptySoftware.header = "No software detected";
  }

  if (isSoftwareDisabled) {
    emptySoftware.header = "Software inventory disabled";
    emptySoftware.info = (
      <>
        Users with the admin role can{" "}
        <CustomLink
          url="https://fleetdm.com/docs/using-fleet/vulnerability-processing#configuration"
          text="turn on software inventory"
          newTab
        />
        .
      </>
    );
  }
  if (softwareFilter === "vulnerableSoftware" && !isSearching) {
    emptySoftware.header = `No software detected`;
  }

  return (
    <EmptyTable
      graphicName="empty-software"
      header={emptySoftware.header}
      info={emptySoftware.info}
    />
  );
};

export default EmptySoftwareTable;
