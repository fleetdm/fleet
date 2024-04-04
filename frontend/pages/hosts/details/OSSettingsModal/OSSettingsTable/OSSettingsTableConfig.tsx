import React from "react";
import { Column } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { IHostMdmData } from "interfaces/host";
import {
  FLEET_FILEVAULT_PROFILE_DISPLAY_NAME,
  // FLEET_FILEVAULT_PROFILE_IDENTIFIER,
  IHostMdmProfile,
  MdmDDMProfileStatus,
  MdmProfileStatus,
  ProfilePlatform,
  isWindowsDiskEncryptionStatus,
} from "interfaces/mdm";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";

import OSSettingStatusCell from "./OSSettingStatusCell";
import { generateWinDiskEncryptionProfile } from "../../helpers";

export const isMdmProfileStatus = (
  status: string
): status is MdmProfileStatus => {
  return status !== "action_required";
};

export interface IHostMdmProfileWithAddedStatus
  extends Omit<IHostMdmProfile, "status"> {
  status: OsSettingsTableStatusValue;
}

type ITableColumnConfig = Column<IHostMdmProfileWithAddedStatus>;
type ITableStringCellProps = IStringCellProps<IHostMdmProfileWithAddedStatus>;

export type INonDDMProfileStatus = MdmProfileStatus | "action_required";

export type OsSettingsTableStatusValue =
  | MdmDDMProfileStatus
  | INonDDMProfileStatus;

/**
 * generates the formatted tooltip for the error column.
 * the expected format of the error string is:
 * "key1: value1, key2: value2, key3: value3"
 */
const generateFormattedTooltip = (detail: string) => {
  const keyValuePairs = detail.split(/, */);
  const formattedElements: JSX.Element[] = [];

  // Special case to handle bitlocker error message. It does not follow the
  // expected string format so we will just render the error message as is.
  if (
    detail.includes("BitLocker") ||
    detail.includes("preparing volume for encryption")
  ) {
    return detail;
  }

  keyValuePairs.forEach((pair, i) => {
    const [key, value] = pair.split(/: */);
    if (key && value) {
      formattedElements.push(
        <span key={key}>
          <b>{key.trim()}:</b> {value.trim()}
          {/* dont add the trailing comma for the last element */}
          {i !== keyValuePairs.length - 1 && (
            <>
              ,<br />
            </>
          )}
        </span>
      );
    }
  });

  return formattedElements.length ? <>{formattedElements}</> : detail;
};

/**
 * generates the error tooltip for the error column. This will be formatted or
 * unformatted.
 */
const generateErrorTooltip = (
  cellValue: string,
  platform: ProfilePlatform,
  detail: string
) => {
  if (platform !== "windows") {
    return cellValue;
  }
  return generateFormattedTooltip(detail);
};

const tableHeaders: ITableColumnConfig[] = [
  {
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ITableStringCellProps) => {
      return <TextCell value={cellProps.cell.value} />;
    },
  },
  {
    Header: "Status",
    disableSortBy: true,
    accessor: "status",
    Cell: (cellProps: ITableStringCellProps) => {
      return (
        <OSSettingStatusCell
          status={cellProps.row.original.status}
          operationType={cellProps.row.original.operation_type}
          profileName={cellProps.row.original.name}
        />
      );
    },
  },
  {
    Header: "Error",
    disableSortBy: true,
    accessor: "detail",
    Cell: (cellProps: ITableStringCellProps): JSX.Element => {
      const profile = cellProps.row.original;

      const value =
        (profile.status === "failed" && profile.detail) ||
        DEFAULT_EMPTY_CELL_VALUE;

      const tooltip =
        profile.status === "failed"
          ? generateErrorTooltip(
              value,
              cellProps.row.original.platform,
              profile.detail
            )
          : null;

      return (
        <TooltipTruncatedTextCell
          tooltipBreakOnWord
          tooltip={tooltip}
          value={value}
        />
      );
    },
  },
];

const makeWindowsRows = ({ profiles, os_settings }: IHostMdmData) => {
  const rows: IHostMdmProfileWithAddedStatus[] = [];

  if (profiles) {
    rows.push(...profiles);
  }

  if (
    os_settings?.disk_encryption?.status &&
    isWindowsDiskEncryptionStatus(os_settings.disk_encryption.status)
  ) {
    rows.push(
      generateWinDiskEncryptionProfile(
        os_settings.disk_encryption.status,
        os_settings.disk_encryption.detail
      )
    );
  }

  if (rows.length === 0 && !profiles) {
    return null;
  }

  return rows;
};

const makeDarwinRows = ({ profiles, macos_settings }: IHostMdmData) => {
  if (!profiles) {
    return null;
  }

  let rows: IHostMdmProfileWithAddedStatus[] = profiles;
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
