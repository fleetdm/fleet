/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";

import { CellProps, Column, HeaderProps } from "react-table";

import DefaultColumnFilter from "components/TableContainer/DataTable/DefaultColumnFilter";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import {
  getSortTypeFromColumnType,
  getUniqueColumnNamesFromRows,
  internallyTruncateText,
} from "utilities/helpers";
import { IQueryTableColumn } from "interfaces/osquery_table";

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
  // TODO - narrow typing down this entire chain of logic
  // typed as any[] to accomodate loose typing of websocket API
  results: T[], // {col:val, ...} for each row of query results
  tableColumns?: IQueryTableColumn[] | []
): Column<T>[] => {
  const uniqueColumnNames = getUniqueColumnNamesFromRows(results);
  const columnsConfigs = uniqueColumnNames.map<Column<T>>((colName) => {
    return {
      id: colName as string,
      Header: (headerProps: HeaderProps<T>) => (
        <HeaderCell
          value={headerProps.column.id}
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      ),
      accessor: colName,
      Cell: (cellProps: CellProps<T>) => {
        const val = cellProps?.cell?.value;
        return !!val?.length && val.length > 300
          ? internallyTruncateText(val)
          : val ?? null;
      },
      Filter: DefaultColumnFilter,
      disableSortBy: false,
      sortType: getSortTypeFromColumnType(colName, tableColumns),
    };
  });
  return _unshiftHostname(columnsConfigs);
};

export default generateColumnConfigsFromRows;
