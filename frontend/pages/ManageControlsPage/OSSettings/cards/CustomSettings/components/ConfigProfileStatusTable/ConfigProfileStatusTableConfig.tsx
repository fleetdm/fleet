import React from "react";
import { Column } from "react-table";

import {
  INumberCellProps,
  IStringCellProps,
} from "interfaces/datatable_config";
import { MdmProfileStatus } from "interfaces/mdm";
import { IGetConfigProfileStatusResponse } from "services/entities/config_profiles";

import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
import { IndicatorStatus } from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";

import ConfigProfileHostCountCell from "../ConfigProfileHostCountCell";

interface IConfigProfileRowData {
  status: string;
  hosts: number;
}

// This is the order in which the statuses will be displayed in the table. It
// will always be in this order.
const STATUS_ORDER = ["verified", "verifying", "pending", "failed"];

export interface IStatusCellValue {
  displayName: string;
  statusName: IndicatorStatus;
}

const STATUS_DISPLAY_OPTIONS: Record<MdmProfileStatus, IStatusCellValue> = {
  verified: {
    displayName: "Verified",
    statusName: "success",
  },
  verifying: {
    displayName: "Verifying",
    statusName: "successPartial",
  },
  pending: {
    displayName: "Pending",
    statusName: "pendingPartial",
  },
  failed: {
    displayName: "Failed",
    statusName: "error",
  },
};

type IConfigProfileStatusColumnConfig = Column<IConfigProfileRowData>;
type IStatusCellProps = IStringCellProps<IConfigProfileRowData>;
type IHostCellProps = INumberCellProps<IConfigProfileRowData>;

export const generateTableConfig = (
  teamId: number,
  uuid: string,
  profileStatus: IGetConfigProfileStatusResponse,
  onClickResend: (hostCount: number, status: string) => void
): IConfigProfileStatusColumnConfig[] => {
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
            status={statusOption.statusName}
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
          <ConfigProfileHostCountCell
            teamId={teamId}
            count={cell.value}
            uuid={uuid}
            status={cell.row.original.status}
            onClickResend={() =>
              onClickResend(cell.value, cell.row.original.status)
            }
          />
        );
      },
    },
  ];
};

export const generateTableData = (
  profileStatus: IGetConfigProfileStatusResponse
): IConfigProfileRowData[] => {
  const tableData = STATUS_ORDER.map((status) => ({
    status,
    hosts: profileStatus[
      status as keyof IGetConfigProfileStatusResponse
    ] as number,
  }));

  return tableData;
};
