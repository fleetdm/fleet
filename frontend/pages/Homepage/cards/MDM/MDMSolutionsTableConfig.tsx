import React from "react";
import { Link } from "react-router";

import { IMdmSolution } from "interfaces/macadmins";

import PATHS from "router/paths";
import { greyCell } from "utilities/helpers";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import Chevron from "../../../../../assets/images/icon-chevron-right-9x6@2x.png";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IMdmSolution;
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

const solutionsTableHeaders = [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ICellProps) => (
      <TextCell
        greyed={greyCell(cellProps.cell.value)}
        value={cellProps.cell.value}
      />
    ),
  },
  {
    title: "Server URL",
    Header: "Server URL",
    disableSortBy: true,
    accessor: "server_url",
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
    accessor: "hosts_count",
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
          to={`${PATHS.MANAGE_HOSTS}?mdm_id=${cellProps.row.original.id}`}
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

export const generateSolutionsTableHeaders = (): IDataColumn[] => {
  return solutionsTableHeaders;
};

const enhanceSolutionsData = (solutions: IMdmSolution[]): IMdmSolution[] => {
  return Object.values(solutions).map((solution) => {
    return {
      id: solution.id,
      name: solution.name || "Unknown",
      server_url: solution.server_url,
      hosts_count: solution.hosts_count,
    };
  });
};

export const generateSolutionsDataSet = (
  solutions: IMdmSolution[] | null
): IMdmSolution[] => {
  if (!solutions) {
    return [];
  }
  return [...enhanceSolutionsData(solutions)];
};

export default { generateSolutionsTableHeaders, generateSolutionsDataSet };
