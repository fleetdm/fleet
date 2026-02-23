import React from "react";
import { ISoftware } from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";

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

const generateTableHeaders = (teamId?: number): IDataColumn[] => [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ICellProps) => (
      <TooltipTruncatedTextCell
        value={cellProps.cell.value}
        className="w150"
        key={cellProps.cell.value}
      />
    ),
  },
  {
    title: "Version",
    Header: "Version",
    disableSortBy: true,
    accessor: "version",
    Cell: (cellProps: ICellProps) => (
      <TooltipTruncatedTextCell
        value={cellProps.cell.value}
        className="w150"
        key={`${cellProps.row.original.name}-${cellProps.cell.value}`}
      />
    ),
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
        <ViewAllHostsLink
          queryParams={{ software_id: cellProps.cell.value, team_id: teamId }} // TODO: Should redirect with the current team id?
          className="software-link"
          condensed
        />
      );
    },
  },
];

export default generateTableHeaders;
