// This component is used on DashboardPage.tsx > Software.tsx,
// Host Details / Device User > Software.tsx, and SoftwarePage.tsx

import React from "react";

import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";
import { IEmptyTableProps } from "interfaces/empty_table";
import { ISoftwareDropdownFilterVal } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/helpers";

export interface IEmptySoftwareTableProps {
  softwareFilter?: ISoftwareDropdownFilterVal;
  isSoftwareDisabled?: boolean;
  isCollectingSoftware?: boolean;
  isSearching?: boolean;
}

const generateTypeText = (softwareFilter?: ISoftwareDropdownFilterVal) => {
  if (softwareFilter === "installableSoftware") {
    return "installable";
  }
  return softwareFilter === "vulnerableSoftware" ? "vulnerable" : "";
};

const EmptySoftwareTable = ({
  softwareFilter,
  isSoftwareDisabled,
  isCollectingSoftware,
  isSearching,
}: IEmptySoftwareTableProps): JSX.Element => {
  const softwareTypeText = generateTypeText(softwareFilter);

  const emptySoftware: IEmptyTableProps = {
    header: `No ${softwareTypeText} software match the current search criteria`,
    info:
      "This report is updated every hour to protect the performance of your devices.",
  };

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
  if (softwareFilter === "vulnerableSoftware" && !isSearching) {
    emptySoftware.header = "No vulnerable software detected";
    emptySoftware.info =
      "This report is updated every hour to protect the performance of your devices.";
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
