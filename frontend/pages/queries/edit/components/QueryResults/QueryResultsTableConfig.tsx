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
  // TODO - narrow typing down this entire chain of logic
  // typed as any[] to accomodate loose typing of websocket API
  results: any[] // {col:val, ...} for each row of query results
): Column[] => {
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
        const val = cellProps?.cell?.value;
        return !!val?.length && val.length > 300
          ? internallyTruncateText(val)
          : val ?? null;
      },
      Filter: DefaultColumnFilter,
      disableSortBy: false,
    };
  });
  return _unshiftHostname(columns);
};

export default generateColumnsFromRows;
