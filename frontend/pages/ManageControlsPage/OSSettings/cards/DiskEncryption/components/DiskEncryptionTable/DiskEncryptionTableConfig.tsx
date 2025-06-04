import React from "react";

import { DiskEncryptionStatus } from "interfaces/mdm";
import {
  IDiskEncryptionStatusAggregate,
  IDiskEncryptionSummaryResponse,
} from "services/entities/disk_encryption";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import { IndicatorStatus } from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";

export interface IStatusCellValue {
  displayName: string;
  statusName: IndicatorStatus;
  value: DiskEncryptionStatus;
  tooltip?: string | JSX.Element;
}

interface IStatusCellProps {
  cell: {
    value: IStatusCellValue;
  };
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: {
      status: IStatusCellValue;
      teamId: number;
    };
  };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

type IDataColumn = {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IStatusCellProps) => JSX.Element);
};

const defaultTableHeaders: IDataColumn[] = [
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "status",
    Cell: ({ cell: { value } }: IStatusCellProps) => {
      const tooltipProp = value.tooltip
        ? { tooltipText: value.tooltip }
        : undefined;
      return (
        <StatusIndicatorWithIcon
          status={value.statusName}
          value={value.displayName}
          tooltip={tooltipProp}
        />
      );
    },
  },
  {
    title: "macOS hosts",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
        disableSortBy
      />
    ),
    disableSortBy: true,
    accessor: "macosHosts",
    Cell: ({ cell: { value: aggregateCount } }: ICellProps) => {
      return (
        <div className="disk-encryption-table__aggregate-table-data">
          <TextCell value={aggregateCount} />
        </div>
      );
    },
  },
  {
    title: "Windows hosts",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
        disableSortBy
      />
    ),
    disableSortBy: true,
    accessor: "windowsHosts",
    Cell: ({ cell: { value: aggregateCount } }: ICellProps) => {
      return <TextCell value={aggregateCount} />;
    },
  },
  {
    title: "Linux hosts",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
        disableSortBy
      />
    ),
    disableSortBy: true,
    accessor: "linuxHosts",
    Cell: ({ cell: { value: aggregateCount } }: ICellProps) => {
      return <TextCell value={aggregateCount} />;
    },
  },
  {
    title: "",
    Header: "",
    accessor: "linkToFilteredHosts",
    disableSortBy: true,
    Cell: (cellProps: ICellProps) => {
      return (
        <>
          {cellProps.row.original && (
            <ViewAllHostsLink className="view-hosts-link" rowHover noLink />
          )}
        </>
      );
    },
  },
];

export const generateTableHeaders = (): IDataColumn[] => {
  return defaultTableHeaders;
};

const STATUS_CELL_VALUES: Record<DiskEncryptionStatus, IStatusCellValue> = {
  verified: {
    displayName: "Verified",
    statusName: "success",
    value: "verified",
    tooltip:
      "These hosts turned disk encryption on and sent their key to Fleet. Fleet verified with osquery.",
  },
  verifying: {
    displayName: "Verifying",
    statusName: "successPartial",
    value: "verifying",
    tooltip:
      "These hosts acknowledged the MDM command to turn on disk encryption. Fleet is verifying with " +
      "osquery and retrieving the disk encryption key. This may take up to one hour.",
  },
  action_required: {
    displayName: "Action required (pending)",
    statusName: "pendingPartial",
    value: "action_required",
    tooltip: (
      <>
        Ask the end user to follow <b>Disk encryption</b> instructions on their{" "}
        <b>My device</b> page.
      </>
    ),
  },
  enforcing: {
    displayName: "Enforcing (pending)",
    statusName: "pendingPartial",
    value: "enforcing",
    tooltip:
      "These hosts will receive the MDM command to turn on disk encryption when the hosts come online.",
  },
  failed: {
    displayName: "Failed",
    statusName: "error",
    value: "failed",
  },
  removing_enforcement: {
    displayName: "Removing enforcement (pending)",
    statusName: "pendingPartial",
    value: "removing_enforcement",
    tooltip:
      "These hosts will receive the MDM command to turn off disk encryption when the hosts come online.",
  },
};

// Order of the status column. We want the order to always be the same.
const STATUS_ORDER = [
  "verified",
  "verifying",
  "failed",
  "action_required",
  "enforcing",
  "removing_enforcement",
] as const;

export const generateTableData = (
  data?: IDiskEncryptionSummaryResponse,
  currentTeamId?: number
) => {
  if (!data) return [];

  const rowFromStatusEntry = (
    status: DiskEncryptionStatus,
    statusAggregate: IDiskEncryptionStatusAggregate
  ) => ({
    status: STATUS_CELL_VALUES[status],
    macosHosts: statusAggregate.macos,
    windowsHosts: statusAggregate.windows,
    linuxHosts: statusAggregate.linux,
    teamId: currentTeamId,
  });

  return STATUS_ORDER.map((status) => rowFromStatusEntry(status, data[status]));
};
