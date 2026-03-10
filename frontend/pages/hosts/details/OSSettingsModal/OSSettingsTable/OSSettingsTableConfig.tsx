import React from "react";
import { Column } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { HostAndroidCertStatus, IHostMdmData } from "interfaces/host";
import {
  FLEET_FILEVAULT_PROFILE_DISPLAY_NAME,
  IHostMdmProfile,
  isLinuxDiskEncryptionStatus,
  isWindowsDiskEncryptionStatus,
  MdmDDMProfileStatus,
  MdmProfileStatus,
} from "interfaces/mdm";
import { isDDMProfile } from "services/entities/mdm";
import { isAppleDevice, isIPadOrIPhone } from "interfaces/platform";

import OSSettingsNameCell from "./OSSettingsNameCell";
import OSSettingStatusCell from "./OSSettingStatusCell";
import OSSettingsErrorCell from "./OSSettingsErrorCell";

import {
  generateLinuxDiskEncryptionSetting,
  generateRecoveryLockPasswordSetting,
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
  | INonDDMProfileStatus
  | HostAndroidCertStatus;

const generateTableConfig = (
  canResendProfiles: boolean,
  resendRequest: (profileUUID: string) => Promise<void>,
  onProfileResent: () => void,
  canRotateRecoveryLockPassword?: boolean,
  rotateRecoveryLockPassword?: () => Promise<void>
): ITableColumnConfig[] => {
  return [
    {
      Header: "Name",
      disableSortBy: true,
      accessor: "name",
      Cell: (cellProps: ITableStringCellProps) => {
        let scope = cellProps.row.original.scope;

        if (isIPadOrIPhone(cellProps.row.original.platform)) {
          scope = null; // Don't show user-scoped icon for iOS/iPadOS profiles, since we don't support user channels.
        }

        return (
          <OSSettingsNameCell
            profileName={cellProps.cell.value}
            scope={scope}
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
            profileUUID={cellProps.row.original.profile_uuid}
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

        const isAppleMobileConfigProfile =
          isAppleDevice(platform) && !isDDMProfile(cellProps.row.original);
        const isWindowsProfile = platform === "windows";

        const isRecoveryLockRow =
          cellProps.row.original.profile_uuid === "rec_lock_dummy";

        return (
          <OSSettingsErrorCell
            canResendProfiles={
              canResendProfiles &&
              (isWindowsProfile || isAppleMobileConfigProfile)
            }
            canRotateRecoveryLockPassword={
              isRecoveryLockRow && canRotateRecoveryLockPassword
            }
            profile={cellProps.row.original}
            resendRequest={resendRequest}
            rotateRecoveryLockPassword={rotateRecoveryLockPassword}
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

const makeDarwinRows = ({
  profiles,
  macos_settings,
  os_settings,
}: IHostMdmData) => {
  let rows: IHostMdmProfileWithAddedStatus[] = profiles ?? [];

  if (macos_settings?.disk_encryption === "action_required") {
    const dERow = profiles?.find(
      (p) => p.name === FLEET_FILEVAULT_PROFILE_DISPLAY_NAME
    );
    if (dERow) {
      // a reference to the original object in rows, so successfully updates it
      dERow.status = "action_required";
    }
  }

  if (os_settings?.recovery_lock_password?.status) {
    rows = [
      ...rows,
      generateRecoveryLockPasswordSetting(
        os_settings.recovery_lock_password.status,
        os_settings.recovery_lock_password.detail
      ),
    ];
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
