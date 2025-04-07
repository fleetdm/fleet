/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";

import { CellProps, Column, HeaderProps } from "react-table";

import DefaultColumnFilter from "components/TableContainer/DataTable/DefaultColumnFilter";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import {
  getUniqueColsAreNumTypeFromRows,
  internallyTruncateText,
} from "utilities/helpers";

const _unshiftHostname = <T extends object>(columns: Column<T>[]) => {
  const newHeaders = [...columns];
  const displayNameIndex = columns.findIndex(
    (h) => h.id === "host_display_name"
  );
  if (displayNameIndex >= 0) {
    // remove hostname header from headers
    const [displayNameHeader] = newHeaders.splice(displayNameIndex, 1);
    // reformat title and insert at start of headers array
    newHeaders.unshift({ ...displayNameHeader, id: "Host" });
  }
  // TODO: Remove after v5 when host_hostname is removed rom API response.
  const hostNameIndex = columns.findIndex((h) => h.id === "host_hostname");
  if (hostNameIndex >= 0) {
    newHeaders.splice(hostNameIndex, 1);
  }
  // end remove
  return newHeaders;
};

const generateColumnConfigsFromRows = <T extends Record<keyof T, unknown>>(
  results: T[] // {col:val, ...} for each row of query results
): Column<T>[] => {
  const colsAreNumTypes = getUniqueColsAreNumTypeFromRows(results) as Map<
    string,
    boolean
  >;
  const columnConfigs = Array.from(colsAreNumTypes.keys()).map<Column<T>>(
    (colName) => {
      return {
        id: colName,
        Header: (headerProps: HeaderProps<T>) => (
          <HeaderCell
            value={headerProps.column.id}
            isSortedDesc={headerProps.column.isSortedDesc}
          />
        ),
        // generic for convenience, can assume keyof T is a string
        accessor: (data) => data[colName as keyof T],
        Cell: (cellProps: CellProps<T>) => {
          const val = cellProps?.cell?.value;
          return !!val?.length && val.length > 300
            ? internallyTruncateText(val)
            : val ?? null;
        },
        Filter: DefaultColumnFilter,
        disableSortBy: false,
        sortType: colsAreNumTypes.get(colName)
          ? "alphanumeric"
          : "caseInsensitive",
      };
    }
  );
  return _unshiftHostname(columnConfigs);
};

export default generateColumnConfigsFromRows;
