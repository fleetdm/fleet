import React from "react";
import { Column } from "react-table";

import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
import {
  INumberCellProps,
  IStringCellProps,
} from "interfaces/datatable_config";
import { IGetConfigProfileStatusResponse } from "services/entities/config_profiles";

type IConfigProfileStatus = "verified" | "verifying" | "pending" | "failed";

interface IConfigProfileRowData {
  status: string;
  hosts: number;
}

const STAUTS_ORDER = ["verified", "verifying", "pending", "failed"];

export interface IStatusCellValue {
  displayName: string;
  statusName: IConfigProfileStatus;
  value: IConfigProfileStatus;
}

const STATUS_DISPLAY_OPTIONS = {
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
} as const;

type IConfigProfileStatusColumnConfig = Column<IConfigProfileRowData>;
type IStatusCellProps = IStringCellProps<IConfigProfileRowData>;
type IHostCellProps = INumberCellProps<IConfigProfileRowData>;

export const generateTableConfig = (
  profileStatus: IGetConfigProfileStatusResponse,
  onClickResend: (status: IConfigProfileStatus) => void
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
        return <span>{cell.value}</span>;
      },
    },
  ];
};

export const generateTableData = (
  profileStatus: IGetConfigProfileStatusResponse
): IConfigProfileRowData[] => {
  const tableData = STAUTS_ORDER.map((status) => ({
    status,
    hosts: profileStatus[
      status as keyof IGetConfigProfileStatusResponse
    ] as number,
  }));

  return tableData;
};
