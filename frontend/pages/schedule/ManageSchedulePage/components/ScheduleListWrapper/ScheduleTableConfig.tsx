/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { performanceIndicator, secondsToDhms } from "fleet/helpers";

// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import TextCell from "components/TableContainer/DataTable/TextCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";
import PillCell from "components/TableContainer/DataTable/PillCell";
import { IDropdownOption } from "interfaces/dropdownOption";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";
import { ITeamScheduledQuery } from "interfaces/team_scheduled_query";

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
    original: IGlobalScheduledQuery | ITeamScheduledQuery;
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
interface IAllScheduledQueryTableData {
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
    all_scheduled_query: IGlobalScheduledQuery | ITeamScheduledQuery
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
      accessor: "query_name",
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
      title: "Performance impact",
      Header: "Performance impact",
      disableSortBy: true,
      accessor: "performance",
      Cell: (cellProps) => <PillCell value={cellProps.cell.value} />,
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

const generateInheritedQueriesTableHeaders = (): IDataColumn[] => {
  return [
    {
      title: "Query",
      Header: "Query",
      disableSortBy: true,
      accessor: "query_name",
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
      title: "Performance impact",
      Header: "Performance impact",
      disableSortBy: true,
      accessor: "performance",
      Cell: (cellProps) => <PillCell value={cellProps.cell.value} />,
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

const enhanceAllScheduledQueryData = (
  all_scheduled_queries: IGlobalScheduledQuery[] | ITeamScheduledQuery[],
  teamId: number | undefined
): IAllScheduledQueryTableData[] => {
  return all_scheduled_queries.map(
    (all_scheduled_query: IGlobalScheduledQuery | ITeamScheduledQuery) => {
      const scheduledQueryPerformance = {
        user_time_p50: all_scheduled_query.stats?.user_time_p50,
        system_time_p50: all_scheduled_query.stats?.system_time_p50,
        total_executions: all_scheduled_query.stats?.total_executions,
      };
      return {
        name: all_scheduled_query.name,
        query_name: all_scheduled_query.query_name,
        interval: all_scheduled_query.interval,
        actions: generateActionDropdownOptions(),
        id: all_scheduled_query.id,
        query_id: all_scheduled_query.query_id,
        snapshot: all_scheduled_query.snapshot,
        removed: all_scheduled_query.removed,
        platform: all_scheduled_query.platform,
        version: all_scheduled_query.version,
        shard: all_scheduled_query.shard,
        type: teamId ? "team_scheduled_query" : "global_scheduled_query",
        performance: [
          performanceIndicator(scheduledQueryPerformance),
          all_scheduled_query.id,
        ],
      };
    }
  );
};

const generateDataSet = (
  all_scheduled_queries: IGlobalScheduledQuery[],
  teamId: number | undefined
): IAllScheduledQueryTableData[] => {
  return [...enhanceAllScheduledQueryData(all_scheduled_queries, teamId)];
};

export {
  generateInheritedQueriesTableHeaders,
  generateTableHeaders,
  generateDataSet,
};
