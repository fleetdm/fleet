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
  /** noSearchQuery is true when there is no search string filtering the results */
  noSearchQuery?: boolean;
  /** isCollectingSoftware is only used on the Dashboard page with a TODO to revisit */
  isCollectingSoftware?: boolean;
  /** true if the team has any software installers or VPP apps available to install on hosts */
  installableSoftwareExists?: boolean;
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
  noSearchQuery,
  isCollectingSoftware,
  installableSoftwareExists,
}: IEmptySoftwareTableProps): JSX.Element => {
  const softwareTypeText = generateTypeText(tableName, softwareFilter);

  const emptySoftware: IEmptyTableProps = {
    header: "No items match the current search criteria",
    info: `Expecting to see ${softwareTypeText}? Check back later.`,
  };

  if (noSearchQuery && softwareFilter === "allSoftware") {
    emptySoftware.header = `No ${tableName} detected`;
  }

  if (softwareFilter === "allSoftware" && installableSoftwareExists) {
    emptySoftware.header = `No ${tableName} detected`;
    emptySoftware.info = "Install software on your hosts to see versions.";
  }

  if (isCollectingSoftware) {
    emptySoftware.header = `No ${tableName} detected`;
    emptySoftware.info = `Expecting to see ${softwareTypeText}? Check back later.`;
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
      graphicName="empty-search-question"
      header={emptySoftware.header}
      info={emptySoftware.info}
    />
  );
};

export default EmptySoftwareTable;
