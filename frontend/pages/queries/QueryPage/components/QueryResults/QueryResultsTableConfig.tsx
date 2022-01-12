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
import { ICampaignQueryResult } from "interfaces/campaign";

import DefaultColumnFilter from "components/TableContainer/DataTable/DefaultColumnFilter";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

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
  const i = headers.findIndex((h) => h.id === "host_hostname");
  if (i >= 0) {
    // remove hostname header from headers
    const [hostnameHeader] = newHeaders.splice(i, 1);
    // reformat title and insert at start of headers array
    newHeaders.unshift({ ...hostnameHeader, title: "hostname" });
  }
  return newHeaders;
};

const resultsTableHeaders = (results: ICampaignQueryResult[]): Column[] => {
  // Derive the table headers based on the shape of the first result
  // TODO: Investigate how best to detect and handle cases where the results are not all the same shape
  const keys = results[0] ? Object.keys(results[0]) : [];
  const headers = keys.map((key) => {
    return {
      id: key,
      title: key,
      Header: (headerProps: IHeaderProps) => (
        <HeaderCell
          value={headerProps.column.title || headerProps.column.id}
          isSortedDesc={headerProps.column.isSortedDesc}
          disableSortBy
        />
      ),
      accessor: key,
      Cell: (cellProps: ICellProps) => cellProps?.cell?.value || null,
      Filter: DefaultColumnFilter,
      // filterType: "text",
      disableSortBy: true,
    };
  });
  return _unshiftHostname(headers);
};

export default resultsTableHeaders;
