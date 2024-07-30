import React from "react";

import { IMdmSolution } from "interfaces/mdm";

import TextCell from "components/TableContainer/DataTable/TextCell";
import InternalLinkCell from "../../../../components/TableContainer/DataTable/InternalLinkCell";
import { IMdmSolutionTableData } from "./MDM";

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

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableGlobalFilter?: boolean;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

export const generateSolutionsTableHeaders = (): IDataColumn[] => [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "displayName",
    Cell: (cellProps: ICellProps) => (
      <InternalLinkCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Hosts",
    Header: "Hosts",
    disableSortBy: true,
    accessor: "hosts_count",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
];

export const generateSolutionsDataSet = (
  solutions: IMdmSolutionTableData[]
): IMdmSolutionTableData[] => {
  return solutions.map((solution) => {
    return {
      ...solution,
      displayName: solution.name || "Unknown",
    };
  });
};
