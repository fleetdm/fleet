/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { find } from "lodash";

import {
  getPerformanceImpactDescription,
  secondsToDhms,
} from "utilities/helpers";
import { IScheduledQuery } from "interfaces/scheduled_query";
import { IDropdownOption } from "interfaces/dropdownOption";

import Checkbox from "components/forms/fields/Checkbox";
import ActionsDropdown from "components/ActionsDropdown";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import PerformanceImpactCell from "components/TableContainer/DataTable/PerformanceImpactCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";

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

interface IRowProps {
  row: {
    original: IScheduledQuery;
    getToggleRowSelectedProps: () => IGetToggleAllRowsSelectedProps;
    toggleRowSelected: () => void;
  };
}

interface ICellProps extends IRowProps {
  cell: {
    value: string | number | boolean;
  };
}

interface IPerformanceImpactCellProps extends IRowProps {
  cell: {
    value: { indicator: string; id: number };
  };
}

interface IActionsDropdownProps extends IRowProps {
  cell: {
    value: IDropdownOption[];
  };
}

interface IDataColumn {
  id?: string;
  title?: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor?: string;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IPerformanceImpactCellProps) => JSX.Element)
    | ((props: IActionsDropdownProps) => JSX.Element);
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

interface IPackQueriesTableData extends IScheduledQuery {
  logging_string: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  actionSelectHandler: (value: string, scheduled_query: IScheduledQuery) => void
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
      title: "Query",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Frequency",
      Header: "Frequency",
      disableSortBy: false,
      accessor: "interval",
      Cell: (cellProps: ICellProps) => (
        <TextCell
          formatter={(val) => secondsToDhms(val)}
          value={cellProps.cell.value}
        />
      ),
    },
    {
      title: "Platform",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "platform_string",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Logging",
      Header: "Logging",
      disableSortBy: false,
      accessor: "logging_string",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      Header: () => {
        return (
          <div>
            <TooltipWrapper
              tipContent={
                <>
                  This is the average performance
                  <br />
                  impact across all hosts where
                  <br />
                  this query was scheduled.
                </>
              }
            >
              Performance impact
            </TooltipWrapper>
          </div>
        );
      },
      disableSortBy: true,
      accessor: "performance",
      Cell: (cellProps: IPerformanceImpactCellProps) => (
        <PerformanceImpactCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps: IActionsDropdownProps) => (
        <ActionsDropdown
          options={cellProps.cell.value}
          onChange={(value: string) =>
            actionSelectHandler(value, cellProps.row.original)
          }
          placeholder="Actions"
        />
      ),
    },
  ];
};

const generateLoggingTypeString = (
  snapshot: boolean,
  removed: boolean
): string => {
  if (snapshot) {
    return "Snapshot";
  }

  // Default is differential with removals, so we treat null as removed = true
  if (removed !== false) {
    return "Differential";
  }

  return "Differential (ignore removal)";
};

const generatePlatformTypeString = (platforms: string | undefined): string => {
  const ALL_PLATFORMS = [
    { text: "All", value: "all" },
    { text: "Windows", value: "windows" },
    { text: "Linux", value: "linux" },
    { text: "macOS", value: "darwin" },
  ];

  if (platforms) {
    const platformsArray = platforms.split(",");

    const textArray = platformsArray.map((platform) => {
      // Trim spaces from the platform
      const trimmedPlatform = platform.trim();
      const platformObject = find(ALL_PLATFORMS, { value: trimmedPlatform });
      // Convert trimmed value to the corresponding text if the value exists
      // in the ALL_PLATFORMS array.
      // Otherwise, just use the trimmed value.
      const text = platformObject ? platformObject.text : trimmedPlatform;

      return text;
    });

    const displayText = textArray.join(", ");

    return displayText;
  }

  return "All";
};

const generateVersionString = (version: string | undefined): string => {
  if (version) {
    return version;
  }
  return "Any";
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

const enhancePackQueriesData = (
  packQueries: IScheduledQuery[]
): IPackQueriesTableData[] => {
  return packQueries.map((query) => {
    const scheduledQueryPerformance = {
      user_time_p50: query.stats?.user_time_p50,
      system_time_p50: query.stats?.system_time_p50,
      total_executions: query.stats?.total_executions,
    };
    return {
      id: query.id,
      name: query.query_name,
      interval: query.interval,
      pack_id: query.pack_id,
      platform: query.platform || undefined,
      query: query.query,
      query_id: query.query_id,
      removed: query.removed,
      snapshot: query.snapshot,
      logging_string: generateLoggingTypeString(query.snapshot, query.removed),
      platform_string: generatePlatformTypeString(query.platform),
      shard: query.shard,
      version: query.version,
      versionString: generateVersionString(query.version),
      created_at: query.created_at,
      updated_at: query.updated_at,
      query_name: query.query_name,
      actions: generateActionDropdownOptions(),
      performance: [
        getPerformanceImpactDescription(scheduledQueryPerformance),
        query.query_id,
      ],
      stats: query.stats,
    };
  });
};

const generateDataSet = (
  queries: IScheduledQuery[]
): IPackQueriesTableData[] => {
  // Cannot pass undefined to enhancePackQueriesData
  if (!queries) {
    return queries;
  }

  return [...enhancePackQueriesData(queries)];
};

export { generateTableHeaders, generateDataSet };
