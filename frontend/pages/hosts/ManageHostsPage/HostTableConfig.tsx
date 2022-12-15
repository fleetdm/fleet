/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { Column } from "react-table";
import ReactTooltip from "react-tooltip";

import { IDeviceUser, IHost } from "interfaces/host";
import Checkbox from "components/forms/fields/Checkbox";
import DiskSpaceGraph from "components/DiskSpaceGraph";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import IssueCell from "components/TableContainer/DataTable/IssueCell/IssueCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import StatusIndicator from "components/StatusIndicator";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import {
  humanHostMemory,
  humanHostLastRestart,
  humanHostLastSeen,
  humanHostDetailUpdated,
  hostTeamName,
} from "utilities/helpers";
import { IConfig } from "interfaces/config";
import { IDataColumn } from "interfaces/datatable_config";
import { ITeamSummary } from "interfaces/team";
import { IUser } from "interfaces/user";
import PATHS from "router/paths";
import permissionUtils from "utilities/permissions";
import getHostStatusTooltipText from "../helpers";

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
    original: IHost;
    getToggleRowSelectedProps: () => IGetToggleAllRowsSelectedProps;
    toggleRowSelected: () => void;
  };
}

interface INumberCellProps {
  cell: {
    value: number;
  };
  row: {
    original: IHost;
    getToggleRowSelectedProps: () => IGetToggleAllRowsSelectedProps;
    toggleRowSelected: () => void;
  };
}

interface IDeviceUserCellProps {
  cell: {
    value: IDeviceUser[];
  };
  row: {
    original: IHost;
  };
}

const condenseDeviceUsers = (users: IDeviceUser[]): string[] => {
  if (!users?.length) {
    return [];
  }
  const condensed =
    users.length === 4
      ? users
          .slice(-4)
          .map((u) => u.email)
          .reverse()
      : users
          .slice(-3)
          .map((u) => u.email)
          .reverse() || [];
  return users.length > 4
    ? condensed.concat(`+${users.length - 3} more`) // TODO: confirm limit
    : condensed;
};

const tooltipTextWithLineBreaks = (lines: string[]) => {
  return lines.map((line) => {
    return (
      <span key={Math.random().toString().slice(2)}>
        {line}
        <br />
      </span>
    );
  });
};

const lastSeenTime = (status: string, seenTime: string): string => {
  if (status !== "online") {
    return `Last Seen: ${humanHostLastSeen(seenTime)} UTC`;
  }
  return "Online";
};

const allHostTableHeaders: IDataColumn[] = [
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
    title: "Host",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "display_name",
    Cell: (cellProps: ICellProps) => (
      <LinkCell
        value={cellProps.cell.value}
        path={PATHS.HOST_DETAILS(cellProps.row.original.id)}
        title={lastSeenTime(
          cellProps.row.original.status,
          cellProps.row.original.seen_time
        )}
      />
    ),
    disableHidden: true,
  },
  {
    title: "Hostname",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hostname",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Computer name",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "computer_name",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Team",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "team_name",
    Cell: (cellProps: ICellProps) => (
      <TextCell value={cellProps.cell.value} formatter={hostTeamName} />
    ),
  },
  {
    title: "Status",
    Header: (headerProps: IHeaderProps): JSX.Element => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={`
             Online hosts will respond to a live query. Offline<br/>
             hosts won’t respond to a live query because<br/>
             they may be shut down, asleep, or not<br/>
             connected to the internet.`}
        >
          Status
        </TooltipWrapper>
      );
      return <HeaderCell value={titleWithToolTip} disableSortBy />;
    },
    disableSortBy: true,
    accessor: "status",
    Cell: (cellProps: ICellProps) => {
      const value = cellProps.cell.value;
      const tooltip = {
        id: cellProps.row.original.id,
        tooltipText: getHostStatusTooltipText(value),
      };
      return <StatusIndicator value={value} tooltip={tooltip} />;
    },
  },
  {
    title: "Issues",
    Header: "Issues",
    disableSortBy: true,
    accessor: "issues",
    Cell: (cellProps: ICellProps) => (
      <IssueCell
        issues={cellProps.row.original.issues}
        rowId={cellProps.row.original.id}
      />
    ),
  },
  {
    title: "Disk space available",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "gigs_disk_space_available",
    Cell: (cellProps: INumberCellProps): JSX.Element => {
      const {
        id,
        platform,
        percent_disk_space_available,
      } = cellProps.row.original;
      return (
        <DiskSpaceGraph
          baseClass="gigs_disk_space_available__cell"
          gigsDiskSpaceAvailable={cellProps.cell.value}
          percentDiskSpaceAvailable={percent_disk_space_available}
          id={`disk-space__${id}`}
          platform={platform}
        />
      );
    },
  },
  {
    title: "Operating system",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "os_version",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Osquery",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "osquery_version",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Used by",
    Header: "Used by",
    disableSortBy: true,
    accessor: "device_mapping",
    Cell: (cellProps: IDeviceUserCellProps): JSX.Element => {
      const numUsers = cellProps.cell.value?.length || 0;
      const users = condenseDeviceUsers(cellProps.cell.value || []);
      if (users.length) {
        const tooltipText = tooltipTextWithLineBreaks(users);
        return (
          <>
            <span
              className={`text-cell ${
                users.length > 1 ? "text-muted tooltip" : ""
              }`}
              data-tip
              data-for={`device_mapping__${cellProps.row.original.id}`}
              data-tip-disable={users.length <= 1}
            >
              {numUsers === 1 ? users[0] : `${numUsers} users`}
            </span>
            <ReactTooltip
              effect="solid"
              backgroundColor="#3e4771"
              id={`device_mapping__${cellProps.row.original.id}`}
              data-html
            >
              <span className={`tooltip__tooltip-text`}>{tooltipText}</span>
            </ReactTooltip>
          </>
        );
      }
      return <span className="text-muted">---</span>;
    },
  },
  {
    title: "Private IP address",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "primary_ip",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Public IP address",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "public_ip",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Last fetched",
    Header: (headerProps: IHeaderProps): JSX.Element => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={`
            The last time the host<br/> reported vitals.
          `}
        >
          Last fetched
        </TooltipWrapper>
      );
      return (
        <HeaderCell
          value={titleWithToolTip}
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      );
    },
    accessor: "detail_updated_at",
    Cell: (cellProps: ICellProps) => (
      <TextCell
        value={cellProps.cell.value}
        formatter={humanHostDetailUpdated}
      />
    ),
  },
  {
    title: "Last seen",
    Header: (headerProps: IHeaderProps): JSX.Element => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={`
            The last time the <br/>host was online.
          `}
        >
          Last seen
        </TooltipWrapper>
      );
      return (
        <HeaderCell
          value={titleWithToolTip}
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      );
    },
    accessor: "seen_time",
    Cell: (cellProps: ICellProps) => (
      <TextCell value={cellProps.cell.value} formatter={humanHostLastSeen} />
    ),
  },
  {
    title: "UUID",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "uuid",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Last restarted",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "uptime",
    Cell: (cellProps: ICellProps) => {
      const { uptime, detail_updated_at } = cellProps.row.original;

      return (
        <TextCell value={humanHostLastRestart(detail_updated_at, uptime)} />
      );
    },
  },
  {
    title: "CPU",
    Header: "CPU",
    disableSortBy: true,
    accessor: "cpu_type",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "RAM",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "memory",
    Cell: (cellProps: ICellProps) => (
      <TextCell value={cellProps.cell.value} formatter={humanHostMemory} />
    ),
  },
  {
    title: "MAC address",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "primary_mac",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Serial number",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hardware_serial",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Hardware model",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hardware_model",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
];

const defaultHiddenColumns = [
  "hostname",
  "computer_name",
  "device_mapping",
  "primary_mac",
  "public_ip",
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
  currentTeam: ITeamSummary | undefined
): IDataColumn[] => {
  return allHostTableHeaders.reduce(
    (columns: Column[], currentColumn: Column) => {
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
  currentTeam: ITeamSummary | undefined
): IDataColumn[] => {
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
