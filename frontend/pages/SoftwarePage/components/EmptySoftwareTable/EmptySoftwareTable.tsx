// This component is used on DashboardPage.tsx > Software.tsx,
// Host Details / Device User > Software.tsx, and SoftwarePage.tsx

import React from "react";

import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";
import { IEmptyTableProps } from "interfaces/empty_table";
import { ISoftwareDropdownFilterVal } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/helpers";

export interface IEmptySoftwareTableProps {
  softwareFilter?: ISoftwareDropdownFilterVal;
  /** tableName is displayed in the search empty state */
  tableName?: string;
  isSoftwareDisabled?: boolean;
  /** isNotDetectingSoftware renders empty states when no search string is present */
  isNotDetectingSoftware?: boolean;
  /** isCollectingSoftware is only used on the Dashboard page with a TODO to revisit */
  isCollectingSoftware?: boolean;
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
  softwareFilter = "allSoftware",
  tableName = "software",
  isSoftwareDisabled,
  isNotDetectingSoftware,
  isCollectingSoftware,
}: IEmptySoftwareTableProps): JSX.Element => {
  const softwareTypeText = generateTypeText(tableName, softwareFilter);

  const emptySoftware: IEmptyTableProps = {
    header: "No items match the current search criteria",
    info: `Expecting to see ${softwareTypeText}? Check back later.`,
  };

  if (isNotDetectingSoftware && softwareFilter === "allSoftware") {
    emptySoftware.header = "No software detected";
  }

  if (isCollectingSoftware) {
    emptySoftware.header = "No software detected";
    emptySoftware.info =
      "This report is updated every hour to protect the performance of your devices.";
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

  return (
    <EmptyTable
      graphicName="empty-software"
      header={emptySoftware.header}
      info={emptySoftware.info}
    />
  );
};

export default EmptySoftwareTable;
