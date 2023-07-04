import React from "react";

import {
  formatOperatingSystemDisplayName,
  IOperatingSystemVersion,
} from "interfaces/operating_system";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";

interface ICellProps {
  cell: {
    value: string;
  };
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

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

const generateDefaultTableHeaders = (teamId?: number): IDataColumn[] => [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name_only",
    Cell: ({ cell: { value } }: ICellProps) => (
      <TextCell
        value={value}
        formatter={(name) => formatOperatingSystemDisplayName(name)}
      />
    ),
  },
  {
    title: "Version",
    Header: "Version",
    disableSortBy: true,
    accessor: "version",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Hosts",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    disableSortBy: false,
    accessor: "hosts_count",
    Cell: (cellProps: ICellProps): JSX.Element => {
      const { hosts_count, name_only, version } = cellProps.row.original;
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
              className="os-hosts-link"
            />
          </span>
        </span>
      );
    },
  },
];

const generateTableHeaders = (
  includeName: boolean,
  teamId?: number
): IDataColumn[] => {
  if (!includeName) {
    return generateDefaultTableHeaders(teamId).filter(
      (column) => column.title !== "Name"
    );
  }
  return generateDefaultTableHeaders(teamId);
};

export default generateTableHeaders;
