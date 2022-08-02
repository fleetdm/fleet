import React from "react";

import { IMacadminsResponse } from "interfaces/host";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IMacadminsResponse;
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

const munkiIssuesTableHeaders = [
  {
    title: "Issue",
    Header: (headerProps: IHeaderProps): JSX.Element => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={`
            Issues reported the last time Munki ran on each host.
          `}
        >
          Issue
        </TooltipWrapper>
      );
      return <HeaderCell value={titleWithToolTip} disableSortBy />;
    },
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Type",
    Header: "Type",
    disableSortBy: true,
    accessor: "type",
    Cell: (cellProps: ICellProps) => (
      <TextCell
        value={
          cellProps.cell.value.charAt(0).toUpperCase() +
          cellProps.cell.value.slice(1)
        }
      />
    ),
  },
  {
    title: "Hosts",
    Header: (headerProps: IHeaderProps): JSX.Element => {
      return (
        <HeaderCell
          value={"Hosts"}
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      );
    },
    disableSortBy: false,
    accessor: "hosts_count",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
];

const generateMunkiIssuesTableHeaders = (): IDataColumn[] => {
  return munkiIssuesTableHeaders;
};

export default generateMunkiIssuesTableHeaders;
