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
  // Table headers are derived from the shape of the first result.
  // Note: It is possible that results may vary from the shape of the first result.
  // For example, different versions of osquery may have new columns in a table
  // However, this is believed to be a very unlikely scenario and there have been
  // no reported issues.
  const keys = results[0] ? Object.keys(results[0]) : [];
  const headers = keys.map((key) => {
    return {
      id: key,
      title: key,
      Header: (headerProps: IHeaderProps) => (
        <HeaderCell
          value={headerProps.column.title || headerProps.column.id}
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      ),
      accessor: key,
      Cell: (cellProps: ICellProps) => cellProps?.cell?.value || null,
      Filter: DefaultColumnFilter,
      // filterType: "text",
    };
  });
  return _unshiftHostname(headers);
};

export default resultsTableHeaders;
