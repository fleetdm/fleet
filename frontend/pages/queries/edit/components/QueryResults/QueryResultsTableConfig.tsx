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
import { internallyTruncateText } from "utilities/helpers";

type IHeaderProps = HeaderProps<TableInstance> & {
  column: ColumnInstance & IDataColumn;
};

type ICellProps = CellProps<TableInstance>;

interface IDataColumn extends ColumnInterface {
  title?: string;
  accessor: string;
}

const _unshiftHostname = (columns: IDataColumn[]) => {
  const newHeaders = [...columns];
  const displayNameIndex = columns.findIndex(
    (h) => h.id === "host_display_name"
  );
  if (displayNameIndex >= 0) {
    // remove hostname header from headers
    const [displayNameHeader] = newHeaders.splice(displayNameIndex, 1);
    // reformat title and insert at start of headers array
    newHeaders.unshift({ ...displayNameHeader, title: "Host" });
  }
  // TODO: Remove after v5 when host_hostname is removed rom API response.
  const hostNameIndex = columns.findIndex((h) => h.id === "host_hostname");
  if (hostNameIndex >= 0) {
    newHeaders.splice(hostNameIndex, 1);
  }
  // end remove
  return newHeaders;
};

const generateColumnsFromRows = (
  results: any[] // {col:val, ...} for each row
): Column[] => {
  /* Results include an array of objects, each representing a table row
  Each key value pair in an object represents a column name and value
  To create headers, use JS set to create an array of all unique column names */
  const uniqueColumnNames = Array.from(
    results.reduce(
      (accOuter, row) =>
        Object.keys(row).reduce(
          (accInner, colNameInRow) => accInner.add(colNameInRow),
          accOuter
        ),
      new Set() // Set prevents listing duplicate headers
    )
  );

  const columns = uniqueColumnNames.map((colName) => {
    return {
      id: colName as string,
      title: colName as string,
      Header: (headerProps: IHeaderProps) => (
        <HeaderCell
          value={headerProps.column.title || headerProps.column.id}
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      ),
      accessor: colName as string,
      Cell: (cellProps: ICellProps) => {
        if (cellProps?.cell?.value) {
          const val = cellProps.cell.value;
          return val.length !== undefined && val.length > 300
            ? internallyTruncateText(cellProps.cell.value)
            : cellProps.cell.value;
        }
        return null;
      },
      Filter: DefaultColumnFilter,
      disableSortBy: false,
    };
  });
  return _unshiftHostname(columns);
};

export default generateColumnsFromRows;
