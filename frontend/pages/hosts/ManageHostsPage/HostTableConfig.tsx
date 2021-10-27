/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";

import { IHost } from "interfaces/host";
import Checkbox from "components/forms/fields/Checkbox";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import IssueCell from "components/TableContainer/DataTable/IssueCell/IssueCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import StatusCell from "components/TableContainer/DataTable/StatusCell/StatusCell";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import {
  humanHostMemory,
  humanHostUptime,
  humanHostLastSeen,
  humanHostDetailUpdated,
  hostTeamName,
} from "fleet/helpers";
import { IConfig } from "interfaces/config";
import { ITeam } from "interfaces/team";
import { IUser } from "interfaces/user";
import PATHS from "router/paths";
import permissionUtils from "utilities/permissions";
import IssueIcon from "../../../../assets/images/icon-issue-fleet-black-16x16@2x.png";

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
    value: string;
  };
  row: {
    original: IHost;
    getToggleRowSelectedProps: () => any; // TODO: do better with types
    toggleRowSelected: () => void;
  };
}

interface IHostDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell: (props: ICellProps) => JSX.Element;
  id?: string;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

const lastSeenTime = (status: string, seenTime: string): string => {
  if (status !== "online") {
    return `Last Seen: ${humanHostLastSeen(seenTime)} UTC`;
  }
  return "Online";
};

const allHostTableHeaders: IHostDataColumn[] = [
  // We are using React Table useRowSelect functionality for the selection header.
  // More information on its API can be found here
  // https://react-table.tanstack.com/docs/api/useRowSelect
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
    title: "Hostname",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hostname",
    Cell: (cellProps) => (
      <LinkCell
        value={cellProps.cell.value}
        path={PATHS.HOST_DETAILS(cellProps.row.original)}
        title={lastSeenTime(
          cellProps.row.original.status,
          cellProps.row.original.seen_time
        )}
      />
    ),
    disableHidden: true,
  },
  {
    title: "Team",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "team_name",
    Cell: (cellProps) => (
      <TextCell value={cellProps.cell.value} formatter={hostTeamName} />
    ),
  },
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "status",
    Cell: (cellProps) => <StatusCell value={cellProps.cell.value} />,
  },
  {
    title: "Issues",
    Header: () => <img alt="host issues" src={IssueIcon} />,
    disableSortBy: true,
    accessor: "issues",
    Cell: (cellProps) => (
      <IssueCell
        issues={cellProps.row.original.issues}
        rowId={cellProps.row.original.id}
      />
    ),
  },
  {
    title: "OS",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "os_version",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Osquery",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "osquery_version",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "IP address",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "primary_ip",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Last fetched",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "detail_updated_at",
    Cell: (cellProps) => (
      <TextCell
        value={cellProps.cell.value}
        formatter={humanHostDetailUpdated}
      />
    ),
  },
  {
    title: "Last seen",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "seen_time",
    Cell: (cellProps) => (
      <TextCell value={cellProps.cell.value} formatter={humanHostLastSeen} />
    ),
  },
  {
    title: "UUID",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "uuid",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Uptime",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "uptime",
    Cell: (cellProps) => (
      <TextCell value={cellProps.cell.value} formatter={humanHostUptime} />
    ),
  },
  {
    title: "CPU",
    Header: "CPU",
    disableSortBy: true,
    accessor: "cpu_type",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "RAM",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "memory",
    Cell: (cellProps) => (
      <TextCell value={cellProps.cell.value} formatter={humanHostMemory} />
    ),
  },
  {
    title: "MAC address",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "primary_mac",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Serial number",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hardware_serial",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Hardware model",
    Header: (cellProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hardware_model",
    Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
  },
];

const defaultHiddenColumns = [
  "primary_mac",
  "cpu_type",
  "memory",
  "uptime",
  "uuid",
  "seen_time",
  "hardware_model",
  "hardware_serial",
];

/**
 * Will generate a host table column configuration based off of the current user
 * permissions and license tier of fleet they are on.
 */
const generateAvailableTableHeaders = (
  config: IConfig,
  currentUser: IUser,
  currentTeam: ITeam | undefined
): IHostDataColumn[] => {
  return allHostTableHeaders.reduce(
    (columns: IHostDataColumn[], currentColumn: IHostDataColumn) => {
      // skip over column headers that are not shown in free observer tier
      if (
        permissionUtils.isFreeTier(config) &&
        permissionUtils.isGlobalObserver(currentUser)
      ) {
        if (
          currentColumn.accessor === "team_name" ||
          currentColumn.id === "selection"
        ) {
          return columns;
        }
        // skip over column headers that are not shown in free admin/maintainer
      } else if (permissionUtils.isFreeTier(config)) {
        if (currentColumn.accessor === "team_name") {
          return columns;
        }
      } else if (
        // In premium tier, we want to check user role to enable/disable select column
        !permissionUtils.isGlobalAdmin(currentUser) &&
        !permissionUtils.isGlobalMaintainer(currentUser) &&
        !permissionUtils.isTeamMaintainer(
          currentUser,
          currentTeam?.id || null
        ) &&
        !permissionUtils.isTeamAdmin(currentUser, currentTeam?.id || null)
      ) {
        if (currentColumn.id === "selection") {
          return columns;
        }
      }

      columns.push(currentColumn);
      return columns;
    },
    []
  );
};

/**
 * Will generate a host table column configuration that a user currently sees.
 *
 */
const generateVisibleTableColumns = (
  hiddenColumns: string[],
  config: IConfig,
  currentUser: IUser,
  currentTeam: ITeam | undefined
): IHostDataColumn[] => {
  // remove columns set as hidden by the user.
  return generateAvailableTableHeaders(config, currentUser, currentTeam).filter(
    (column) => {
      return !hiddenColumns.includes(column.accessor as string);
    }
  );
};

export {
  defaultHiddenColumns,
  generateAvailableTableHeaders,
  generateVisibleTableColumns,
};
