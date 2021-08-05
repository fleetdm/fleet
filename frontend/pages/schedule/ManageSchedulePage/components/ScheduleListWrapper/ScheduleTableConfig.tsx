/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { secondsToDhms } from "fleet/helpers";

// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import TextCell from "components/TableContainer/DataTable/TextCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";
import { IDropdownOption } from "interfaces/dropdownOption";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
  getToggleAllRowsSelectedProps: () => any; // TODO: do better with types
  toggleAllRowsSelected: () => void;
}

interface ICellProps {
  cell: {
    value: any;
  };
  row: {
    original: IGlobalScheduledQuery;
    getToggleRowSelectedProps: () => any; // TODO: do better with types
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
interface IGlobalScheduledQueryTableData {
  name: string;
  interval: number;
  actions: IDropdownOption[];
  id: number;
  type: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  actionSelectHandler: (
    value: string,
    global_scheduled_query: IGlobalScheduledQuery
  ) => void
): IDataColumn[] => {
  return [
    {
      id: "selection",
      Header: (cellProps: IHeaderProps): JSX.Element => {
        const props = cellProps.getToggleAllRowsSelectedProps();
        const checkboxProps = {
          value: props.checked,
          indeterminate: props.indeterminate,
          onChange: () => cellProps.toggleAllRowsSelected(),
        };
        return <Checkbox {...checkboxProps} />;
      },
      Cell: (cellProps: ICellProps): JSX.Element => {
        const props = cellProps.row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: props.checked,
          onChange: () => cellProps.row.toggleRowSelected(),
        };
        return <Checkbox {...checkboxProps} />;
      },
      disableHidden: true,
    },
    {
      title: "Query",
      Header: "Query",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Frequency",
      Header: "Frequency",
      disableSortBy: true,
      accessor: "interval",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={secondsToDhms(cellProps.cell.value)} />
      ),
    },
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps) => (
        <DropdownCell
          options={cellProps.cell.value}
          onChange={(value: string) =>
            actionSelectHandler(value, cellProps.row.original)
          }
          placeholder={"Actions"}
        />
      ),
    },
  ];
};

const generateActionDropdownOptions = (): IDropdownOption[] => {
  const dropdownOptions = [
    {
      label: "Edit",
      disabled: false,
      value: "edit",
    },
    {
      label: "Remove",
      disabled: false,
      value: "remove",
    },
  ];
  return dropdownOptions;
};

const enhanceGlobalScheduledQueryData = (
  global_scheduled_queries: IGlobalScheduledQuery[]
): IGlobalScheduledQueryTableData[] => {
  return global_scheduled_queries.map((global_scheduled_query) => {
    return {
      name: global_scheduled_query.name,
      interval: global_scheduled_query.interval,
      actions: generateActionDropdownOptions(),
      id: global_scheduled_query.id,
      query_id: global_scheduled_query.query_id,
      snapshot: global_scheduled_query.snapshot,
      removed: global_scheduled_query.removed,
      platform: global_scheduled_query.platform,
      version: global_scheduled_query.version,
      shard: global_scheduled_query.shard,
      type: "global_scheduled_query",
    };
  });
};

const generateDataSet = (
  global_scheduled_queries: IGlobalScheduledQuery[]
): IGlobalScheduledQueryTableData[] => {
  return [...enhanceGlobalScheduledQueryData(global_scheduled_queries)];
};

export { generateTableHeaders, generateDataSet };
