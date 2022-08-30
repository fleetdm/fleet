import React from "react";
import ReactTooltip from "react-tooltip";

import { formatDistanceToNow } from "date-fns";

import PATHS from "router/paths";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { IMunkiIssue } from "interfaces/host";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}
interface ICellProps {
  cell: {
    value: number | string | string[];
  };
  row: {
    original: IMunkiIssue;
    index: number;
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
  Cell: (props: IStringCellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  disableGlobalFilter?: boolean;
  sortType?: string;
  // Filter can be used by react-table to render a filter input inside the column header
  Filter?: () => null | JSX.Element;
  filter?: string; // one of the enumerated `filterTypes` for react-table
  // (see https://github.com/tannerlinsley/react-table/blob/master/src/filterTypes.js)
  // or one of the custom `filterTypes` defined for the `useTable` instance (see `DataTable`)
}

interface IMunkiIssueTableData extends IMunkiIssue {
  time: string;
}

export const generateMunkiIssuesTableData = (
  munkiIssues: IMunkiIssue[] | undefined
): IMunkiIssueTableData[] => {
  if (!munkiIssues) {
    return [];
  }
  return munkiIssues.map((i) => {
    return {
      ...i,
      time: i.created_at,
    };
  });
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateMunkiIssuesTableHeaders = (
  deviceUser = false
): IDataColumn[] => {
  const tableHeaders: IDataColumn[] = [
    {
      title: "Issue",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      disableSortBy: false,
      disableGlobalFilter: false,
      Cell: (cellProps: IStringCellProps) => {
        return <TextCell value={cellProps.cell.value} />;
      },
      sortType: "caseInsensitive",
    },
    {
      title: "Type",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      disableSortBy: true,
      disableGlobalFilter: true,
      accessor: "type",
      Cell: (cellProps: IStringCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Time",
      Header: "Time",
      accessor: "time",
      disableSortBy: false,
      disableGlobalFilter: false,
      Filter: () => null, // input for this column filter outside of column header
      filter: "hasLength", // filters out rows where vulnerabilities has no length if filter value is `true`
      Cell: (cellProps: IStringCellProps) => {
        return <TextCell value={cellProps.cell.value} />;
      },
    },
  ];

  // Device user cannot view all hosts software
  if (deviceUser) {
    tableHeaders.pop();
  }

  return tableHeaders;
};

export default {
  generateMunkiIssuesTableHeaders,
  generateMunkiIssuesTableData,
};
