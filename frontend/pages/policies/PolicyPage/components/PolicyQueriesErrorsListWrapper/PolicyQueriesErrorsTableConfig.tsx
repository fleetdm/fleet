/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { memoize } from "lodash";

// @ts-ignore
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import { IHostPolicyQueryError } from "interfaces/host";
import sortUtils from "utilities/sort";

// TODO functions for paths math e.g., path={PATHS.MANAGE_HOSTS + getParams(cellProps.row.original)}

interface IHeaderProps {
  column: {
    host: string;
    isSortedDesc: boolean;
  };
}

interface ICellProps {
  cell: {
    value: any;
  };
  row: {
    original: IHostPolicyQueryError;
  };
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell: (props: ICellProps) => JSX.Element;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (): IDataColumn[] => {
  const tableHeaders: IDataColumn[] = [
    {
      title: "Host",
      Header: "Host",
      disableSortBy: true,
      accessor: "host_hostname",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "OSQuery Version",
      Header: "OSQuery Version",
      disableSortBy: true,
      accessor: "osquery_version",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Error",
      Header: "Error",
      disableSortBy: true,
      accessor: "error",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
  ];
  return tableHeaders;
};

const generateDataSet = memoize(
  (
    policyHostsErrorsList: IHostPolicyQueryError[] = []
  ): IHostPolicyQueryError[] => {
    policyHostsErrorsList = policyHostsErrorsList.sort((a, b) =>
      sortUtils.caseInsensitiveAsc(a.host_hostname, b.host_hostname)
    );
    return policyHostsErrorsList;
  }
);

export { generateTableHeaders, generateDataSet };
