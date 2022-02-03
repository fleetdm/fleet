/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { find } from "lodash";

import { performanceIndicator } from "fleet/helpers";
import Checkbox from "components/forms/fields/Checkbox";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import PillCell from "components/TableContainer/DataTable/PillCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { IScheduledQuery } from "interfaces/scheduled_query";
import { IDropdownOption } from "interfaces/dropdownOption";

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
    original: IScheduledQuery;
    getToggleRowSelectedProps: () => any; // TODO: do better with types
    toggleRowSelected: () => void;
  };
}

interface IDataColumn {
  id?: string;
  title?: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor?: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

interface IPackQueriesTableData extends IScheduledQuery {
  loggingTypeString: string;
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
      title: "Query name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Frequency",
      Header: "Frequency",
      disableSortBy: false,
      accessor: "interval",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Platform",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "platformTypeString",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      title: "Logging",
      Header: "Logging",
      disableSortBy: false,
      accessor: "loggingTypeString",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
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
      name: query.name,
      interval: query.interval,
      pack_id: query.pack_id,
      platform: query.platform || undefined,
      query: query.query,
      query_id: query.query_id,
      removed: query.removed,
      snapshot: query.snapshot,
      loggingTypeString: generateLoggingTypeString(
        query.snapshot,
        query.removed
      ),
      platformTypeString: generatePlatformTypeString(query.platform),
      shard: query.shard,
      version: query.version,
      versionString: generateVersionString(query.version),
      created_at: query.created_at,
      updated_at: query.updated_at,
      query_name: query.query_name,
      actions: generateActionDropdownOptions(),
      performance: [
        performanceIndicator(scheduledQueryPerformance),
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
