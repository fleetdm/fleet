import React from "react";
import { Column } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { IHostMdmData } from "interfaces/host";
import {
  FLEET_FILEVAULT_PROFILE_DISPLAY_NAME,
  IHostMdmProfile,
  MdmDDMProfileStatus,
  MdmProfileStatus,
  isLinuxDiskEncryptionStatus,
  isWindowsDiskEncryptionStatus,
} from "interfaces/mdm";
import { isDDMProfile } from "services/entities/mdm";

import OSSettingsNameCell from "./OSSettingsNameCell";
import OSSettingStatusCell from "./OSSettingStatusCell";
import OSSettingsErrorCell from "./OSSettingsErrorCell";

import {
  generateLinuxDiskEncryptionSetting,
  generateWinDiskEncryptionSetting,
} from "../../helpers";

export interface IHostMdmProfileWithAddedStatus
  extends Omit<IHostMdmProfile, "status"> {
  status: OsSettingsTableStatusValue;
}

type ITableColumnConfig = Column<IHostMdmProfileWithAddedStatus>;
type ITableStringCellProps = IStringCellProps<IHostMdmProfileWithAddedStatus>;

/** Non DDM profiles can have an `action_required` as a profile status.  DDM
 * Profiles will never have this status.
 */
export type INonDDMProfileStatus = MdmProfileStatus | "action_required";

export type OsSettingsTableStatusValue =
  | MdmDDMProfileStatus
  | INonDDMProfileStatus;

const generateTableConfig = (
  canResendProfiles: boolean,
  resendRequest: (profileUUID: string) => Promise<void>,
  onProfileResent: () => void
): ITableColumnConfig[] => {
  return [
    {
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps: ITableStringCellProps) => {
        return (
          <OSSettingsNameCell
            profileName={cellProps.cell.value}
            scope={cellProps.row.original.scope}
            managedAccount={cellProps.row.original.managed_local_account}
          />
        );
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
            hostPlatform={cellProps.row.original.platform}
          />
        );
      },
    },
    {
      Header: "Error",
      disableSortBy: true,
      accessor: "detail",
      Cell: (cellProps: ITableStringCellProps) => {
        const { platform } = cellProps.row.original;

        const isMacOSMobileConfigProfile =
          platform === "darwin" && !isDDMProfile(cellProps.row.original);
        return (
          <OSSettingsErrorCell
            canResendProfiles={canResendProfiles && isMacOSMobileConfigProfile}
            profile={cellProps.row.original}
            resendRequest={resendRequest}
            onProfileResent={onProfileResent}
          />
        );
      },
    },
  ];
};

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
      generateWinDiskEncryptionSetting(
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

const makeLinuxRows = ({ profiles, os_settings }: IHostMdmData) => {
  const rows: IHostMdmProfileWithAddedStatus[] = [];

  if (profiles) {
    rows.push(...profiles);
  }

  if (
    os_settings?.disk_encryption?.status &&
    isLinuxDiskEncryptionStatus(os_settings.disk_encryption.status)
  ) {
    rows.push(
      generateLinuxDiskEncryptionSetting(
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
        return { ...p, status: "action_required" };
      }
      return p;
    });
  }

  return rows;
};

export const generateTableData = (
  hostMDMData: IHostMdmData,
  platform: string
) => {
  switch (platform) {
    case "windows":
      return makeWindowsRows(hostMDMData);
    case "darwin":
      return makeDarwinRows(hostMDMData);
    case "ubuntu":
      return makeLinuxRows(hostMDMData);
    case "rhel":
      return makeLinuxRows(hostMDMData);
    case "ios":
    case "ipados":
    case "android":
      return hostMDMData.profiles;
    default:
      return null;
  }
};

export default generateTableConfig;
