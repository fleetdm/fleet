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

const generateColumnConfigs = (rows: any[]) =>
  getUniqueColumnNamesFromRows(rows).map((colName) => {
    return {
      id: colName as string,
      title: colName as string,
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
      accessor: colName as string,
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
    };
  });

export default generateColumnConfigs;
