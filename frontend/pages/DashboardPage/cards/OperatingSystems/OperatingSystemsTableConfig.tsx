import React from "react";
import { Link } from "react-router";
import PATHS from "router/paths";

import {
  formatOperatingSystemDisplayName,
  IOperatingSystemVersion,
} from "interfaces/operating_system";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";

import Chevron from "../../../../../assets/images/icon-chevron-right-blue-16x16@2x.png";

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

const defaultTableHeaders = [
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
            <Link
              to={`${PATHS.MANAGE_HOSTS}?os_name=${encodeURIComponent(
                name_only
              )}&os_version=${encodeURIComponent(version)}`}
              className="hosts-link"
            >
              <span className="link-text">View all hosts</span>
              <img
                alt="link to hosts filtered by operating system ID"
                src={Chevron}
              />
            </Link>
          </span>
        </span>
      );
    },
  },
];

const generateTableHeaders = (): IDataColumn[] => {
  return defaultTableHeaders;
};

export default generateTableHeaders;
