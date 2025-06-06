import React from "react";
import { Column } from "react-table";

import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
import {
  INumberCellProps,
  IStringCellProps,
} from "interfaces/datatable_config";
import { IScriptBatchSummaryResponse } from "services/entities/scripts";
import Button from "components/buttons/Button";

interface IHostCountCellProps {
  status: string;
  count: number;
  onClickCancel: () => void;
}

const HostCountCell = ({
  status,
  count,
  onClickCancel,
}: IHostCountCellProps) => {
  const baseClass = "script-batch-status-host-count-cell";
  return (
    <div className={baseClass}>
      <div>{count}</div>
      {status === "pending" && (
        <Button
          className={`${baseClass}__cancel-button`}
          onClick={onClickCancel}
          variant="text-icon"
        >
          <span>Cancel</span>
        </Button>
      )}
    </div>
  );
};

type IStatus = "ran" | "pending" | "errored";

interface IRowData {
  status: string;
  hosts: number;
}

const STATUS_ORDER = ["ran", "pending", "errored"];

export interface IStatusCellValue {
  displayName: string;
  statusName: IStatus;
  value: IStatus;
}

const STATUS_DISPLAY_OPTIONS = {
  ran: {
    displayName: "Ran",
    indicatorStatus: "success",
  },
  pending: {
    displayName: "Pending",
    indicatorStatus: "pendingPartial",
  },
  errored: {
    displayName: "Error",
    indicatorStatus: "error",
  },
} as const;

type IColumnConfig = Column<IRowData>;
type IStatusCellProps = IStringCellProps<IRowData>;
type IHostCellProps = INumberCellProps<IRowData>;

export const generateTableConfig = (
  onClickCancel: () => void
): IColumnConfig[] => {
  return [
    {
      Header: "Status",
      disableSortBy: true,
      accessor: "status",
      Cell: ({ cell: { value } }: IStatusCellProps) => {
        const statusOption =
          STATUS_DISPLAY_OPTIONS[value as keyof typeof STATUS_DISPLAY_OPTIONS];
        return (
          <StatusIndicatorWithIcon
            status={statusOption.indicatorStatus}
            value={statusOption.displayName}
          />
        );
      },
    },
    {
      Header: "Hosts",
      accessor: "hosts",
      disableSortBy: true,
      Cell: ({ cell }: IHostCellProps) => {
        return (
          <HostCountCell
            count={cell.value}
            status={cell.row.original.status}
            onClickCancel={onClickCancel}
          />
        );
      },
    },
  ];
};

export const generateTableData = (
  statusData: IScriptBatchSummaryResponse
): IRowData[] => {
  const tableData = STATUS_ORDER.map((status) => ({
    status,
    hosts: statusData[status as keyof IScriptBatchSummaryResponse] as number,
  }));

  return tableData;
};
