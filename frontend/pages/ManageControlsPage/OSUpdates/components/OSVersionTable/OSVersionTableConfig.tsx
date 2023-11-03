import React from "react";

import { IOperatingSystemVersion } from "interfaces/operating_system";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import TextCell from "components/TableContainer/DataTable/TextCell";

import OSTypeCell from "../OSTypeCell";
import { IFilteredOperatingSystemVersion } from "../CurrentVersionSection/CurrentVersionSection";

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
          versionName={row.original.name_only}
        />
      ),
    },
    {
      title: "Version",
      Header: "Version",
      disableSortBy: true,
      accessor: "version",
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
        const { hosts_count, name_only, version } = row.original;
        return (
          <span className="hosts-cell__wrapper">
            <span className="hosts-cell__count">
              <TextCell value={hosts_count} />
            </span>
            <span className="hosts-cell__link">
              <ViewAllHostsLink
                queryParams={{
                  os_name: name_only,
                  os_version: version,
                  team_id: teamId,
                }}
                condensed
                className="os-hosts-link"
              />
            </span>
          </span>
        );
      },
    },
  ];
};
