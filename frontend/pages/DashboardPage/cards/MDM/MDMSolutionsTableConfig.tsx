import React from "react";

import { IMdmSolution } from "interfaces/mdm";

import { greyCell } from "utilities/helpers";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties

interface IMDMSolutionWithPlatformId extends IMdmSolution {
  selectedPlatformLabelId?: number;
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IMDMSolutionWithPlatformId;
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
  disableGlobalFilter?: boolean;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

export const generateSolutionsTableHeaders = (
  teamId?: number
): IDataColumn[] => [
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
        <ViewAllHostsLink
          queryParams={{ mdm_id: cellProps.row.original.id, team_id: teamId }}
          className="mdm-solution-link"
          platformLabelId={cellProps.row.original.selectedPlatformLabelId}
        />
      );
    },
    disableHidden: true,
  },
];

const enhanceSolutionsData = (
  solutions: IMdmSolution[],
  selectedPlatformLabelId?: number
): IMdmSolution[] => {
  return Object.values(solutions).map((solution) => {
    return {
      id: solution.id,
      name: solution.name || "Unknown",
      server_url: solution.server_url,
      hosts_count: solution.hosts_count,
      selectedPlatformLabelId,
    };
  });
};

export const generateSolutionsDataSet = (
  solutions: IMdmSolution[] | null,
  selectedPlatformLabelId?: number
): IMdmSolution[] => {
  if (!solutions) {
    return [];
  }
  return [...enhanceSolutionsData(solutions, selectedPlatformLabelId)];
};
