import DefaultColumnFilter from "components/TableContainer/DataTable/DefaultColumnFilter";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import React from "react";

import {
  CellProps,
  ColumnInstance,
  ColumnInterface,
  HeaderProps,
  TableInstance,
} from "react-table";
import {
  getUniqueColumnNamesFromRows,
  humanHostLastSeen,
  internallyTruncateText,
} from "utilities/helpers";

type IHeaderProps = HeaderProps<TableInstance> & {
  column: ColumnInstance & IDataColumn;
};

type ICellProps = CellProps<TableInstance>;

interface IDataColumn extends ColumnInterface {
  title?: string;
  accessor: string;
}

const generateColumnConfigs = (rows: Record<string, string>[]) =>
  // casting necessary because of loose typing of below method
  // see note there for more details
  (getUniqueColumnNamesFromRows(rows) as string[]).map((colName) => {
    return {
      id: colName,
      title: colName,
      Header: (headerProps: IHeaderProps) => (
        <HeaderCell
          value={
            // Sentence case last fetched
            headerProps.column.title === "last_fetched"
              ? "Last fetched"
              : headerProps.column.title || headerProps.column.id
          }
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      ),
      accessor: colName,
      Cell: (cellProps: ICellProps) => {
        // Sorts chronologically by date, but UI displays readable last fetched
        if (cellProps.column.id === "last_fetched") {
          return humanHostLastSeen(cellProps?.cell?.value);
        }
        // truncate columns longer than 300 characters
        const val = cellProps?.cell?.value;
        return !!val?.length && val.length > 300
          ? internallyTruncateText(val)
          : val ?? null;
      },
      Filter: DefaultColumnFilter, // Component hides filter for last_fetched
      filterType: "text",
      disableSortBy: false,
      sortType: "caseInsensitive",
    };
  });

export default generateColumnConfigs;
