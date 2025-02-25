/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";

import { format } from "date-fns";

// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import StatusIndicator from "components/StatusIndicator";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";

import PATHS from "router/paths";

import { IPack } from "interfaces/pack";

interface IGetToggleAllRowsSelectedProps {
  checked: boolean;
  indeterminate: boolean;
  title: string;
  onChange: () => void;
  style: { cursor: string };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
  getToggleAllRowsSelectedProps: () => IGetToggleAllRowsSelectedProps;
  toggleAllRowsSelected: () => void;
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IPack;
    getToggleRowSelectedProps: () => IGetToggleAllRowsSelectedProps;
    toggleRowSelected: () => void;
  };
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell: (props: ICellProps) => JSX.Element;
  id?: string;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

interface IPackTableData {
  id?: number;
  name?: string;
  query_count?: number;
  status?: string;
  total_hosts_count?: number;
  updated_at?: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (): IDataColumn[] => {
  const tableHeaders: IDataColumn[] = [
    {
      id: "selection",
      Header: (cellProps: IHeaderProps): JSX.Element => {
        const props = cellProps.getToggleAllRowsSelectedProps();
        const checkboxProps = {
          value: props.checked,
          indeterminate: props.indeterminate,
          onChange: () => cellProps.toggleAllRowsSelected(),
        };
        return <Checkbox {...checkboxProps} enableEnterToCheck />;
      },
      Cell: (cellProps: ICellProps): JSX.Element => {
        const props = cellProps.row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: props.checked,
          onChange: () => cellProps.row.toggleRowSelected(),
        };
        return <Checkbox {...checkboxProps} enableEnterToCheck />;
      },
      disableHidden: true,
    },
    {
      title: "Name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <LinkCell
          value={cellProps.cell.value}
          path={PATHS.EDIT_PACK(cellProps.row.original.id)}
        />
      ),
    },
    {
      title: "Queries",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "query_count",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Hosts",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "total_hosts_count",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Last modified",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "updated_at",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={format(new Date(cellProps.cell.value), "MM/dd/yy")} />
      ),
    },
    {
      title: "Status",
      Header: "Status",
      disableSortBy: true,
      accessor: "status",
      Cell: (cellProps) => <StatusIndicator value={cellProps.cell.value} />,
    },
  ];
  return tableHeaders;
};

const enhancePackData = (packs: IPack[] | undefined): IPackTableData[] => {
  if (packs) {
    return packs.map((pack: IPack) => {
      return {
        id: pack.id,
        name: pack.name,
        query_count: pack.query_count,
        status: pack.disabled ? "disabled" : "enabled",
        total_hosts_count: pack.total_hosts_count,
        updated_at: pack.updated_at,
      };
    });
  }
  return [];
};

const generateDataSet = (packs: IPack[] | undefined): IPackTableData[] => {
  return [...enhancePackData(packs)];
};

export { generateTableHeaders, generateDataSet };
