import React from "react";

import {
  FileVaultProfileStatus,
  IFileVaultSummaryResponse,
} from "interfaces/mdm";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import { IndicatorStatus } from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";

interface IStatusCellValue {
  displayName: string;
  statusName: IndicatorStatus;
  value: FileVaultProfileStatus;
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
    title: "Hosts",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
        disableSortBy={false}
      />
    ),
    accessor: "hosts",
    Cell: ({
      cell: { value: aggregateCount },
      row: { original },
    }: ICellProps) => {
      return (
        <div className="disk-encryption-table__aggregate-table-data">
          <TextCell value={aggregateCount} formatter={(val) => <>{val}</>} />
          <ViewAllHostsLink
            className="view-hosts-link"
            queryParams={{
              macos_settings_disk_encryption: original.status.value,
              team_id: original.teamId,
            }}
          />
        </div>
      );
    },
  },
];

type StatusNames = keyof IFileVaultSummaryResponse;

type StatusEntry = [StatusNames, number];

export const generateTableHeaders = (): IDataColumn[] => {
  return defaultTableHeaders;
};

const STATUS_CELL_VALUES: Record<FileVaultProfileStatus, IStatusCellValue> = {
  verifying: {
    displayName: "Verifying",
    statusName: "successPartial",
    value: FileVaultProfileStatus.VERIFYING,
    tooltip: "Disk encryption on and key stored in Fleet. Fleet will verify.",
  },
  action_required: {
    displayName: "Action required (pending)",
    statusName: "pendingPartial",
    value: FileVaultProfileStatus.ACTION_REQUIRED,
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
    value: FileVaultProfileStatus.ENFORCING,
    tooltip: "Setting will be enforced when the hosts come online.",
  },
  failed: {
    displayName: "Failed",
    statusName: "error",
    value: FileVaultProfileStatus.FAILED,
  },
  removing_enforcement: {
    displayName: "Removing enforcement (pending)",
    statusName: "pendingPartial",
    value: FileVaultProfileStatus.REMOVING_ENFORCEMENT,
    tooltip: "Enforcement will be removed when the hosts come online.",
  },
};

export const generateTableData = (
  data?: IFileVaultSummaryResponse,
  currentTeamId?: number
) => {
  if (!data) return [];
  const entries = Object.entries(data) as StatusEntry[];

  return entries.map(([status, numHosts]) => ({
    // eslint-disable-next-line object-shorthand
    status: STATUS_CELL_VALUES[status],
    hosts: numHosts,
    teamId: currentTeamId,
  }));
};
