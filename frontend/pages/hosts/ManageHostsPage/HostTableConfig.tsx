/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { CellProps, Column } from "react-table";
import ReactTooltip from "react-tooltip";

import { IDeviceUser, IHost } from "interfaces/host";
import Checkbox from "components/forms/fields/Checkbox";
import DiskSpaceIndicator from "pages/hosts/components/DiskSpaceIndicator";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import HostMdmStatusCell from "components/TableContainer/DataTable/HostMdmStatusCell/HostMdmStatusCell";
import IssueCell from "components/TableContainer/DataTable/IssueCell/IssueCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import StatusIndicator from "components/StatusIndicator";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import NotSupported from "components/NotSupported";

import {
  humanHostMemory,
  humanHostLastSeen,
  hostTeamName,
  tooltipTextWithLineBreaks,
} from "utilities/helpers";
import { COLORS } from "styles/var/colors";
import {
  IHeaderProps,
  IStringCellProps,
  INumberCellProps,
} from "interfaces/datatable_config";
import PATHS from "router/paths";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import getHostStatusTooltipText from "../helpers";

type IHostTableColumnConfig = Column<IHost> & {
  // This is used to prevent these columns from being hidden. This will be
  // used in EditColumnsModal to prevent these columns from being hidden.
  disableHidden?: boolean;
  // We add title in the column config to be able to use it in the EditColumnsModal
  // as well
  title?: string;
};

type IHostTableHeaderProps = IHeaderProps<IHost>;
type IHostTableStringCellProps = IStringCellProps<IHost>;
type IHostTableNumberCellProps = INumberCellProps<IHost>;
type ISelectionCellProps = CellProps<IHost>;
type IIssuesCellProps = CellProps<IHost, IHost["issues"]>;
type IDeviceUserCellProps = CellProps<IHost, IHost["device_mapping"]>;

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

const lastSeenTime = (status: string, seenTime: string): string => {
  if (status !== "online") {
    return `Last Seen: ${humanHostLastSeen(seenTime)} UTC`;
  }
  return "Online";
};

const allHostTableHeaders: IHostTableColumnConfig[] = [
  // We are using React Table useRowSelect functionality for the selection header.
  // More information on its API can be found here
  // https://react-table.tanstack.com/docs/api/useRowSelect
  {
    id: "selection",
    Header: (cellProps: IHostTableHeaderProps) => {
      const props = cellProps.getToggleAllRowsSelectedProps();
      const checkboxProps = {
        value: props.checked,
        indeterminate: props.indeterminate,
        onChange: () => cellProps.toggleAllRowsSelected(),
      };
      return <Checkbox {...checkboxProps} enableEnterToCheck />;
    },
    Cell: (cellProps: ISelectionCellProps) => {
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
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell value="Host" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "display_name",
    id: "display_name",
    Cell: (cellProps: IHostTableStringCellProps) => {
      if (
        // if the host is pending, we want to disable the link to host details
        cellProps.row.original.mdm.enrollment_status === "Pending" &&
        // pending status is only supported for Apple devices
        (cellProps.row.original.platform === "darwin" ||
          cellProps.row.original.platform === "ios" ||
          cellProps.row.original.platform === "ipados") &&
        // osquery version is populated along with the rest of host details so use it
        // here to check if we already have host details and don't need to disable the link
        !cellProps.row.original.osquery_version
      ) {
        return (
          <>
            <span
              className="text-cell"
              data-tip
              data-for={`host__${cellProps.row.original.id}`}
            >
              {cellProps.cell.value}
            </span>
            <ReactTooltip
              effect="solid"
              backgroundColor={COLORS["tooltip-bg"]}
              id={`host__${cellProps.row.original.id}`}
              data-html
            >
              <span className={`tooltip__tooltip-text`}>
                This host was ordered using <br />
                Apple Business Manager <br />
                (ABM). You will see host <br />
                vitals when it is enrolled in Fleet <br />
              </span>
            </ReactTooltip>
          </>
        );
      }
      return (
        <LinkCell
          value={cellProps.cell.value}
          path={PATHS.HOST_DETAILS(cellProps.row.original.id)}
          title={lastSeenTime(
            cellProps.row.original.status,
            cellProps.row.original.seen_time
          )}
        />
      );
    },
    disableHidden: true,
  },
  {
    title: "Hostname",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell
        value="Hostname"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hostname",
    id: "hostname",
    Cell: (cellProps: IHostTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Computer name",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell
        value="Computer name"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "computer_name",
    id: "computer_name",
    Cell: (cellProps: IHostTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Team",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell value="Team" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "team_name",
    id: "team_name",
    Cell: (cellProps) => (
      <TextCell value={cellProps.cell.value} formatter={hostTeamName} />
    ),
  },
  {
    title: "Status",
    Header: (cellProps: IHostTableHeaderProps) => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={
            <>
              Online hosts will respond to a live query. Offline hosts
              won&apos;t respond to a live query because they may be shut down,
              asleep, or not connected to the internet.
            </>
          }
          className="status-header"
        >
          Status
        </TooltipWrapper>
      );
      return (
        <HeaderCell
          value={cellProps.rows.length === 1 ? "Status" : titleWithToolTip}
          disableSortBy
        />
      );
    },
    disableSortBy: true,
    accessor: "status",
    id: "status",
    Cell: (cellProps: IHostTableStringCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      const value = cellProps.cell.value;
      const tooltip = {
        tooltipText: getHostStatusTooltipText(value),
      };
      return <StatusIndicator value={value} tooltip={tooltip} />;
    },
  },
  {
    title: "Issues",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell value="Issues" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "issues",
    id: "issues",
    sortDescFirst: true,
    Cell: (cellProps: IIssuesCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return (
        <IssueCell
          issues={cellProps.row.original.issues}
          rowId={cellProps.row.original.id}
        />
      );
    },
  },
  {
    title: "Disk space available",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell
        value="Disk space available"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "gigs_disk_space_available",
    id: "gigs_disk_space_available",
    Cell: (cellProps: IHostTableNumberCellProps) => {
      const {
        id,
        platform,
        percent_disk_space_available,
      } = cellProps.row.original;
      if (platform === "chrome") {
        return NotSupported;
      }
      return (
        <DiskSpaceIndicator
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
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell
        value="Operating system"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "os_version",
    id: "os_version",
    Cell: (cellProps: IHostTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Osquery",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell
        value="Osquery"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "osquery_version",
    id: "osquery_version",
    Cell: (cellProps: IHostTableStringCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return <TextCell value={cellProps.cell.value} />;
    },
  },
  {
    title: "Used by",
    Header: "Used by",
    disableSortBy: true,
    accessor: "device_mapping",
    id: "device_mapping",
    Cell: (cellProps: IDeviceUserCellProps) => {
      const numUsers = cellProps.cell.value?.length || 0;
      const users = condenseDeviceUsers(cellProps.cell.value || []);
      if (users.length > 1) {
        return (
          <TooltipWrapper
            tipContent={tooltipTextWithLineBreaks(users)}
            underline={false}
            showArrow
            position="top"
            tipOffset={10}
          >
            <TextCell italic value={`${numUsers} users`} />
          </TooltipWrapper>
        );
      }
      if (users.length === 1) {
        return <TextCell value={users[0]} />;
      }
      return <TextCell />;
    },
  },
  {
    title: "Private IP address",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell
        value="Private IP address"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "primary_ip",
    id: "primary_ip",
    Cell: (cellProps: IHostTableStringCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return <TextCell value={cellProps.cell.value} />;
    },
  },
  {
    title: "MDM status",
    Header: () => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={
            <>
              Settings can be updated remotely on hosts with MDM turned
              <br />
              on. To filter by MDM status, head to the Dashboard page.
            </>
          }
        >
          MDM status
        </TooltipWrapper>
      );
      return <HeaderCell value={titleWithToolTip} disableSortBy />;
    },
    disableSortBy: true,
    accessor: (originalRow) => originalRow.mdm.enrollment_status,
    id: "mdm.enrollment_status",
    Cell: HostMdmStatusCell,
  },
  {
    title: "MDM server URL",
    Header: () => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={
            <>
              The MDM server that updates settings on the host. To
              <br />
              filter by MDM server URL, head to the Dashboard page.
            </>
          }
        >
          MDM server URL
        </TooltipWrapper>
      );
      return <HeaderCell value={titleWithToolTip} disableSortBy />;
    },
    disableSortBy: true,
    accessor: (originalRow) => originalRow.mdm.server_url,
    id: "mdm.server_url",
    Cell: (cellProps: IHostTableStringCellProps) => {
      if (cellProps.row.original.platform === "chrome") {
        return NotSupported;
      }
      if (cellProps.cell.value) {
        return <TooltipTruncatedTextCell value={cellProps.cell.value} />;
      }
      return <span className="text-muted">{DEFAULT_EMPTY_CELL_VALUE}</span>;
    },
  },
  {
    title: "Public IP address",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell
        value={
          <TooltipWrapper tipContent="The IP address the host uses to connect to Fleet.">
            Public IP address
          </TooltipWrapper>
        }
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "public_ip",
    id: "public_ip",
    Cell: (cellProps: IHostTableStringCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return (
        <TextCell value={cellProps.cell.value ?? DEFAULT_EMPTY_CELL_VALUE} />
      );
    },
  },
  {
    title: "Last fetched",
    Header: (cellProps: IHostTableHeaderProps) => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={
            <>
              The last time the host
              <br /> reported vitals.
            </>
          }
        >
          Last fetched
        </TooltipWrapper>
      );
      return (
        <HeaderCell
          value={titleWithToolTip}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      );
    },
    accessor: "detail_updated_at",
    id: "detail_updated_at",
    Cell: (cellProps: IHostTableStringCellProps) => (
      <TextCell
        value={{ timeString: cellProps.cell.value }}
        formatter={HumanTimeDiffWithFleetLaunchCutoff}
      />
    ),
  },
  {
    title: "Last seen",
    Header: (cellProps: IHostTableHeaderProps) => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={
            <>
              The last time the <br />
              host was online.
            </>
          }
        >
          Last seen
        </TooltipWrapper>
      );
      return (
        <HeaderCell
          value={titleWithToolTip}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      );
    },
    accessor: "seen_time",
    id: "seen_time",
    Cell: (cellProps: IHostTableStringCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return (
        <TextCell
          value={{ timeString: cellProps.cell.value }}
          formatter={HumanTimeDiffWithFleetLaunchCutoff}
        />
      );
    },
  },
  {
    title: "UUID",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell value="UUID" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "uuid",
    id: "uuid",
    Cell: (cellProps: IHostTableStringCellProps) => (
      <TooltipTruncatedTextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Last restarted",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell
        value="Last restarted"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "last_restarted_at",
    id: "last_restarted_at",
    Cell: (cellProps: IHostTableStringCellProps) => {
      const { platform, last_restarted_at } = cellProps.row.original;

      if (
        platform === "ios" ||
        platform === "ipados" ||
        platform === "chrome"
      ) {
        return NotSupported;
      }
      return (
        <TextCell
          value={{
            timeString: last_restarted_at,
          }}
          formatter={HumanTimeDiffWithFleetLaunchCutoff}
        />
      );
    },
  },
  {
    title: "CPU",
    Header: "CPU",
    disableSortBy: true,
    accessor: "cpu_type",
    id: "cpu_type",
    Cell: (cellProps: IHostTableStringCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return <TextCell value={cellProps.cell.value} />;
    },
  },
  {
    title: "RAM",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell value="RAM" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "memory",
    id: "memory",
    Cell: (cellProps: IHostTableNumberCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return (
        <TextCell value={cellProps.cell.value} formatter={humanHostMemory} />
      );
    },
  },
  {
    title: "MAC address",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell
        value="MAC address"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "primary_mac",
    id: "primary_mac",
    Cell: (cellProps: IHostTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Serial number",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell
        value="Serial number"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hardware_serial",
    id: "hardware_serial",
    Cell: (cellProps: IHostTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Hardware model",
    Header: (cellProps: IHostTableHeaderProps) => (
      <HeaderCell
        value="Hardware model"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hardware_model",
    id: "hardware_model",
    Cell: (cellProps: IHostTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
];

const defaultHiddenColumns = [
  "hostname",
  "computer_name",
  "device_mapping",
  "primary_mac",
  "public_ip",
  "cpu_type",
  // TODO: should those be mdm.<blah>?
  "mdm.server_url",
  "mdm.enrollment_status",
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
const generateAvailableTableHeaders = ({
  isFreeTier = true,
  isOnlyObserver = true,
}: {
  isFreeTier: boolean | undefined;
  isOnlyObserver: boolean | undefined;
}): IHostTableColumnConfig[] => {
  return allHostTableHeaders.reduce(
    (columns: Column<IHost>[], currentColumn: Column<IHost>) => {
      // skip over column headers that are not shown in free observer tier
      if (isFreeTier) {
        if (
          isOnlyObserver &&
          ["selection", "team_name"].includes(currentColumn.id || "")
        ) {
          return columns;
          // skip over column headers that are not shown in free admin/maintainer
        }
        if (
          currentColumn.id === "team_name" ||
          currentColumn.id === "mdm.server_url" ||
          currentColumn.id === "mdm.enrollment_status"
        ) {
          return columns;
        }
      } else if (isOnlyObserver && currentColumn.id === "selection") {
        // In premium tier, we want to check user role to enable/disable select column
        return columns;
      }

      columns.push(currentColumn);
      return columns;
    },
    []
  );
};

/**
 * Will generate a host table column configuration that a user currently sees.
 */
const generateVisibleTableColumns = ({
  hiddenColumns,
  isFreeTier = true,
  isOnlyObserver = true,
}: {
  hiddenColumns: string[];
  isFreeTier: boolean | undefined;
  isOnlyObserver: boolean | undefined;
}): IHostTableColumnConfig[] => {
  // remove columns set as hidden by the user.
  return generateAvailableTableHeaders({ isFreeTier, isOnlyObserver }).filter(
    (column) => {
      return !hiddenColumns.includes(column.id as string);
    }
  );
};

export {
  defaultHiddenColumns,
  generateAvailableTableHeaders,
  generateVisibleTableColumns,
};
