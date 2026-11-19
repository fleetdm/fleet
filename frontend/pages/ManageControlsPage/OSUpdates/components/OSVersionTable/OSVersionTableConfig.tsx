import React from "react";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import { IOperatingSystemVersion } from "interfaces/operating_system";

import { IFilteredOperatingSystemVersion } from "../CurrentVersionSection/CurrentVersionSection";
import OSTypeCell from "../OSTypeCell";

interface IOSTypeCellProps {
  row: {
    original: IFilteredOperatingSystemVersion;
  };
}

interface IHostCellProps {
  row: {
    original: IOperatingSystemVersion;
  };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

// eslint-disable-next-line import/prefer-default-export
export const generateTableHeaders = (teamId: number) => {
  return [
    {
      title: "OS type",
      Header: "OS type",
      disableSortBy: true,
      accessor: "platform",
      Cell: ({ row }: IOSTypeCellProps) => (
        <OSTypeCell
          platform={row.original.platform}
          versionName={row.original.os_name}
        />
      ),
    },
    {
      title: "Version",
      Header: "Version",
      disableSortBy: true,
      accessor: "os_version",
    },
    {
      title: "Hosts",
      accessor: "hosts_count",
      disableSortBy: false,
      Header: (cellProps: IHeaderProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      Cell: ({ row }: IHostCellProps): JSX.Element => {
        const { hosts_count } = row.original;
        return <TextCell value={hosts_count} />;
      },
    },
    {
      title: "",
      Header: "",
      accessor: "linkToFilteredHosts",
      disableSortBy: true,
      Cell: (cellProps: IOSTypeCellProps) => {
        return (
          <>
            {cellProps.row.original && (
              <ViewAllHostsLink
                queryParams={{
                  os_name: cellProps.row.original.os_name,
                  os_version: cellProps.row.original.os_version,
                  fleet_id: teamId,
                }}
                className="os-hosts-link"
                rowHover
              />
            )}
          </>
        );
      },
    },
  ];
};
