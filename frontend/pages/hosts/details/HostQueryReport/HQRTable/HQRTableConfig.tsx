import DefaultColumnFilter from "components/TableContainer/DataTable/DefaultColumnFilter";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import { IHeaderProps, IWebSocketData } from "interfaces/datatable_config";
import React from "react";

import { CellProps, Column } from "react-table";
import {
  getUniqueColsAreNumTypeFromRows,
  humanHostLastSeen,
  internallyTruncateText,
} from "utilities/helpers";

type IHQRTTableColumn = Column<IWebSocketData>;
type ITableHeaderProps = IHeaderProps<IWebSocketData>;
type ITableStringCellProps = CellProps<IWebSocketData, string | unknown>;

const generateColumnConfigs = (rows: IWebSocketData[]): IHQRTTableColumn[] => {
  const colsAreNumTypes = getUniqueColsAreNumTypeFromRows(rows) as Map<
    string,
    boolean
  >;
  return Array.from(colsAreNumTypes.keys()).map<IHQRTTableColumn>((colName) => {
    return {
      id: colName,
      Header: (headerProps: ITableHeaderProps) => (
        <HeaderCell
          value={
            // Sentence case last fetched
            headerProps.column.id === "last_fetched"
              ? "Last fetched"
              : headerProps.column.id || headerProps.column.id
          }
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      ),
      accessor: (data) => data[colName],
      Cell: (cellProps: ITableStringCellProps) => {
        if (typeof cellProps?.cell?.value !== "string") return null;

        // Sorts chronologically by date, but UI displays readable last fetched
        if (cellProps.column.id === "last_fetched") {
          return <>{humanHostLastSeen(cellProps?.cell?.value)}</>;
        }
        // truncate columns longer than 300 characters
        const val = cellProps?.cell?.value;
        return !!val?.length && val.length > 300 ? (
          internallyTruncateText(val)
        ) : (
          <>{val}</>
        );
      },
      Filter: DefaultColumnFilter, // Component hides filter for last_fetched
      filterType: "text",
      disableSortBy: false,
      sortType: "caseInsensitive",
    };
  });
};

export default generateColumnConfigs;
