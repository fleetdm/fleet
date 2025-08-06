import React from "react";
import { Column } from "react-table";

import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
import {
  INumberCellProps,
  IStringCellProps,
} from "interfaces/datatable_config";
import { IScriptBatchSummaryResponseV1 } from "services/entities/scripts";
import ScriptBatchHostCountCell from "../ScriptBatchHostCountCell/ScriptBatchHostCountCell";

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
  batchExecutionId: string,
  onClickCancel: () => void,
  teamId?: number
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
          <ScriptBatchHostCountCell
            count={cell.value}
            status={cell.row.original.status}
            batchExecutionId={batchExecutionId}
            onClickCancel={onClickCancel}
            teamId={teamId}
          />
        );
      },
    },
  ];
};

export const generateTableData = (
  statusData: IScriptBatchSummaryResponseV1
): IRowData[] => {
  const tableData = STATUS_ORDER.map((status) => ({
    status,
    hosts: statusData[status as keyof IScriptBatchSummaryResponseV1] as number,
  }));

  return tableData;
};
