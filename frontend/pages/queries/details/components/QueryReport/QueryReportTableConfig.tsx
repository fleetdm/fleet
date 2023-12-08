/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";

import {
  CellProps,
  Column,
  ColumnInstance,
  ColumnInterface,
  HeaderProps,
  TableInstance,
} from "react-table";

import DefaultColumnFilter from "components/TableContainer/DataTable/DefaultColumnFilter";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import { humanHostLastSeen, internallyTruncateText } from "utilities/helpers";

type IHeaderProps = HeaderProps<TableInstance> & {
  column: ColumnInstance & IDataColumn;
};

type ICellProps = CellProps<TableInstance>;

interface IDataColumn extends ColumnInterface {
  title?: string;
  accessor: string;
}

const _unshiftHostname = (headers: IDataColumn[]) => {
  const newHeaders = [...headers];
  const displayNameIndex = headers.findIndex(
    (h) => h.id === "host_display_name"
  );
  if (displayNameIndex >= 0) {
    // remove hostname header from headers
    const [displayNameHeader] = newHeaders.splice(displayNameIndex, 1);
    // reformat title and insert at start of headers array
    newHeaders.unshift({ ...displayNameHeader, title: "Host" });
  }
  // TODO: Remove after v5 when host_hostname is removed rom API response.
  const hostNameIndex = headers.findIndex((h) => h.id === "host_hostname");
  if (hostNameIndex >= 0) {
    newHeaders.splice(hostNameIndex, 1);
  }
  // end remove
  return newHeaders;
};

const generateReportColumnConfigsFromResults = (results: any[]): Column[] => {
  /* Results include an array of objects, each representing a table row
  Each key value pair in an object represents a column name and value
  To create headers, use JS set to create an array of all unique column names */
  const uniqueColumnNames = Array.from(
    results.reduce(
      (s, o) => Object.keys(o).reduce((t, k) => t.add(k), s),
      new Set() // Set prevents listing duplicate headers
    )
  );

  const columnConfigs = uniqueColumnNames.map((key) => {
    return {
      id: key as string,
      title: key as string,
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
      accessor: key as string,
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
  return _unshiftHostname(columnConfigs);
};

export default generateReportColumnConfigsFromResults;
