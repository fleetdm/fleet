// This component is used on DashboardPage.tsx > Software.tsx, Host Details/Device User > Software.tsx, and ManageSoftwarePage.tsx

import React from "react";

import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";
import { IEmptyTableProps } from "interfaces/empty_table";

export interface IEmptySoftwareTableProps {
  isSoftwareDisabled?: boolean;
  isFilterVulnerable?: boolean;
  isSandboxMode?: boolean;
  isCollectingSoftware?: boolean;
  isSearching?: boolean;
  noSandboxHosts?: boolean;
}

const EmptySoftwareTable = ({
  isSoftwareDisabled,
  isFilterVulnerable,
  isSandboxMode,
  isCollectingSoftware,
  isSearching,
  noSandboxHosts,
}: IEmptySoftwareTableProps): JSX.Element => {
  const emptySoftware: IEmptyTableProps = {
    header: `No ${
      isFilterVulnerable ? "vulnerable " : ""
    }software match the current search criteria`,
    info: `Try again in about ${
      isSandboxMode ? "15 minutes" : "1 hour"
    } as the system catches up.`,
  };
  if (isCollectingSoftware) {
    emptySoftware.header = "No software detected";
    emptySoftware.info =
      "This report is updated every hour to protect the performance of your devices.";
    if (isSandboxMode) {
      emptySoftware.info = noSandboxHosts
        ? "Fleet begins collecting software inventory after a host is enrolled."
        : "Fleet is collecting software inventory";
    }
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
  if (isFilterVulnerable && !isSearching) {
    emptySoftware.header = "No vulnerable software detected";
    emptySoftware.info = `This report is updated every ${
      isSandboxMode ? "15 minutes" : "hour"
    } to protect the performance of your devices.`;
  }

  return (
    <EmptyTable
      iconName="empty-software"
      header={emptySoftware.header}
      info={emptySoftware.info}
    />
  );
};

export default EmptySoftwareTable;
