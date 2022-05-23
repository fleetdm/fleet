/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { memoize } from "lodash";

import { ColumnInstance } from "react-table";

import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import { IHostPolicyQuery } from "interfaces/host";
import sortUtils from "utilities/sort";
import PassIcon from "../../../../../../assets/images/icon-check-circle-green-16x16@2x.png";
import FailIcon from "../../../../../../assets/images/icon-action-fail-16x16@2x.png";

interface IHeaderProps {
  column: ColumnInstance & IDataColumn;
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IHostPolicyQuery;
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
      Header: (headerProps: IHeaderProps): JSX.Element => (
        <HeaderCell
          value={headerProps.column.title || headerProps.column.id}
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      accessor: "hostname",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Status",
      Header: (headerProps: IHeaderProps) => (
        <HeaderCell
          value={headerProps.column.title || headerProps.column.id}
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      ),
      disableSortBy: false,
      sortType: "hasLength",
      accessor: "query_results",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <>
          {cellProps.cell.value.length ? (
            <>
              <img alt="host passing" src={PassIcon} />
              <span className="status-header-text">Yes</span>
            </>
          ) : (
            <>
              <img alt="host passing" src={FailIcon} />
              <span className="status-header-text">No</span>
            </>
          )}
        </>
      ),
    },
  ];
  return tableHeaders;
};

const generateDataSet = memoize(
  (policyHostsList: IHostPolicyQuery[] = []): IHostPolicyQuery[] => {
    policyHostsList = policyHostsList.sort((a, b) =>
      sortUtils.caseInsensitiveAsc(a.hostname, b.hostname)
    );
    return policyHostsList;
  }
);

export { generateTableHeaders, generateDataSet };
