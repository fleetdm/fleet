import TextCell from "components/TableContainer/DataTable/TextCell";
import React from "react";

import { IHostMdmData } from "interfaces/host";
import {
  FLEET_FILEVAULT_PROFILE_DISPLAY_NAME,
  // FLEET_FILEVAULT_PROFILE_IDENTIFIER,
  IHostMdmProfile,
  MdmProfileStatus,
  isWindowsDiskEncryptionStatus,
} from "interfaces/mdm";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import TruncatedTextCell from "components/TableContainer/DataTable/TruncatedTextCell";
import MacSettingStatusCell from "./MacSettingStatusCell";
import { generateWinDiskEncryptionProfile } from "../../helpers";

export interface ITableRowOsSettings extends Omit<IHostMdmProfile, "status"> {
  status: OsSettingsTableStatusValue;
}

export type OsSettingsTableStatusValue = MdmProfileStatus | "action_required";

export const isMdmProfileStatus = (
  status: string
): status is MdmProfileStatus => {
  return status !== "action_required";
};

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: ITableRowOsSettings;
  };
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell: (props: ICellProps) => JSX.Element;
  id?: string;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

const tableHeaders: IDataColumn[] = [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ICellProps): JSX.Element => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "statusText",
    Cell: (cellProps: ICellProps) => {
      return (
        <MacSettingStatusCell
          status={cellProps.row.original.status}
          operationType={cellProps.row.original.operation_type}
          profileName={cellProps.row.original.name}
        />
      );
    },
  },
  {
    title: "Error",
    Header: "Error",
    disableSortBy: true,
    accessor: "detail",
    Cell: (cellProps: ICellProps): JSX.Element => {
      const profile = cellProps.row.original;
      return (
        <TruncatedTextCell
          tooltipBreakOnWord
          value={
            (profile.status === "failed" && profile.detail) ||
            DEFAULT_EMPTY_CELL_VALUE
          }
        />
      );
    },
  },
];

const makeWindowsRows = ({ os_settings }: IHostMdmData) => {
  if (
    !os_settings?.disk_encryption?.status ||
    !isWindowsDiskEncryptionStatus(os_settings.disk_encryption.status)
  ) {
    return null;
  }

  const rows: ITableRowOsSettings[] = [];
  rows.push(
    generateWinDiskEncryptionProfile(
      os_settings.disk_encryption.status,
      os_settings.disk_encryption.detail
    )
  );

  return rows;
};

const makeDarwinRows = ({
  profiles,
  macos_settings,
}: IHostMdmData): ITableRowOsSettings[] | null => {
  if (!profiles) {
    return null;
  }

  let rows: ITableRowOsSettings[] = profiles;
  if (macos_settings?.disk_encryption === "action_required") {
    rows = profiles.map((p) => {
      // TODO: this is a brittle check for the filevault profile
      // it would be better to match on the identifier but it is not
      // currently available in the API response
      if (p.name === FLEET_FILEVAULT_PROFILE_DISPLAY_NAME) {
        return { ...p, status: "action_required" || p.status };
      }
      return p;
    });
  }

  return rows;
};

export const generateTableData = (
  hostMDMData?: IHostMdmData,
  platform?: string
) => {
  if (!platform || !hostMDMData) {
    return null;
  }

  switch (platform) {
    case "windows":
      return makeWindowsRows(hostMDMData);
    case "darwin":
      return makeDarwinRows(hostMDMData);
    default:
      return null;
  }
};

export default tableHeaders;
