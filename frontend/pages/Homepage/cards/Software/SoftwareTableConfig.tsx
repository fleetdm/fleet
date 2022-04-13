import React from "react";
import { Link } from "react-router";

import PATHS from "router/paths";
import { ISoftware } from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";
import Chevron from "../../../../../assets/images/icon-chevron-blue-16x16@2x.png";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: ISoftware;
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

const softwareTableHeaders = [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
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
    Header: "Hosts",
    disableSortBy: true,
    accessor: "hosts_count",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Actions",
    Header: "",
    disableSortBy: true,
    accessor: "id",
    Cell: (cellProps: ICellProps) => {
      return (
        <Link
          to={`${PATHS.MANAGE_HOSTS}?software_id=${cellProps.cell.value}`}
          className="software-link"
        >
          <img alt="link to hosts filtered by software ID" src={Chevron} />
        </Link>
      );
    },
  },
];

const generateTableHeaders = (): IDataColumn[] => {
  return softwareTableHeaders;
};

export default generateTableHeaders;
