import React from "react";
import { Link } from "react-router";

import { IDataTableMDMFormat } from "interfaces/macadmins";

import PATHS from "router/paths";
import TextCell from "components/TableContainer/DataTable/TextCell";
import Chevron from "../../../../../assets/images/icon-chevron-right-9x6@2x.png";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IDataTableMDMFormat;
  };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IStringCellProps extends ICellProps {
  cell: {
    value: string;
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

const enrollmentTableHeaders = [
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "status",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Hosts",
    Header: "Hosts",
    accessor: "hosts",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "",
    Header: "",
    disableSortBy: true,
    disableGlobalFilter: true,
    accessor: "linkToFilteredHosts",
    Cell: (cellProps: IStringCellProps) => {
      return (
        <Link
          to={`${PATHS.MANAGE_HOSTS}?mdm_solution=${cellProps.row.original.status}`}
          className={`mdm-solution-link`}
        >
          View all hosts{" "}
          <img alt="link to hosts filtered by MDM solution" src={Chevron} />
        </Link>
      );
    },
    disableHidden: true,
  },
];

const generateEnrollmentTableHeaders = (): IDataColumn[] => {
  return enrollmentTableHeaders;
};

export default generateEnrollmentTableHeaders;
