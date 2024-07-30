/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { memoize } from "lodash";

import { ColumnInstance } from "react-table";

import Icon from "components/Icon/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import { IPolicyHostResponse } from "interfaces/host";
import sortUtils from "utilities/sort";

interface IHeaderProps {
  column: ColumnInstance & IDataColumn;
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IPolicyHostResponse;
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
      accessor: "display_name",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
      sortType: "caseInsensitive",
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
              <Icon name="success" />
              <span className="status-header-text">Yes</span>
            </>
          ) : (
            <>
              <Icon name="error" />
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
  (policyHostsList: IPolicyHostResponse[] = []): IPolicyHostResponse[] => {
    policyHostsList = policyHostsList.sort((a, b) =>
      sortUtils.caseInsensitiveAsc(a.display_name, b.display_name)
    );
    return policyHostsList;
  }
);

export { generateTableHeaders, generateDataSet };
