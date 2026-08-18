import React from "react";
import { Column, Row } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { HostAndroidCertStatus, IHostMdmData } from "interfaces/host";
import {
  FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
  FLEET_FILEVAULT_PROFILE_DISPLAY_NAME,
  IHostMdmProfile,
  isEnrolledInMdm,
  isLinuxDiskEncryptionStatus,
  isWindowsDiskEncryptionStatus,
  MdmDDMProfileStatus,
  MdmProfileStatus,
  ProfilePlatform,
} from "interfaces/mdm";
import { isDDMProfile } from "services/entities/mdm";
import { isAppleDevice, isIPadOrIPhone } from "interfaces/platform";

import TextCell from "components/TableContainer/DataTable/TextCell";

import OSSettingsNameCell from "./OSSettingsNameCell";
import OSSettingStatusCell from "./OSSettingStatusCell";
import OSSettingsResendCell from "./OSSettingsResendCell";
import { getControlDisplayOption } from "./statusDisplayConfig";

import {
  generateHostNameSettingIfEligible,
  generateLinuxDiskEncryptionSetting,
  generateRecoveryLockPasswordSetting,
  generateWinDiskEncryptionSetting,
  HOST_NAME_SYNTHETIC_PROFILE_UUID,
  LINUX_DISK_ENC_SYNTHETIC_PROFILE_UUID,
  REC_LOCK_SYNTHETIC_PROFILE_UUID,
  WIN_DISK_ENC_SYNTHETIC_PROFILE_UUID,
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

/**
 * Default table order: the controls an admin has to act on first, then the ones
 * still in flight, then the ones already settled. Ranked off the displayed
 * status so a row sorts where the reader sees it, not by the raw API value
 * (several of which share a display name).
 */
const STATUS_SORT_ORDER = [
  "Failed",
  "Action required",
  "Enforcing",
  "Removing enforcement",
  "Verifying",
  "Verified",
];

const getStatusSortRank = (row: IHostMdmProfileWithAddedStatus) => {
  const statusText = getControlDisplayOption(row)?.statusText;
  const rank = statusText ? STATUS_SORT_ORDER.indexOf(statusText) : -1;
  // Unrecognized statuses sort last rather than first.
  return rank === -1 ? STATUS_SORT_ORDER.length : rank;
};

/** Rows Fleet synthesizes from `os_settings` rather than reading out of
 * `mdm.profiles`. No configuration profile stands behind them, so the profile
 * resend endpoint rejects their placeholder UUIDs. */
const SYNTHETIC_PROFILE_UUIDS: string[] = [
  WIN_DISK_ENC_SYNTHETIC_PROFILE_UUID,
  LINUX_DISK_ENC_SYNTHETIC_PROFILE_UUID,
  REC_LOCK_SYNTHETIC_PROFILE_UUID,
  HOST_NAME_SYNTHETIC_PROFILE_UUID,
];

/** Which of the three resend/rotate actions a given row is eligible for, given
 * what the current page and user allow. Shared by the table's action cell and
 * the details modal footer. */
export const getRowActionProps = (
  row: IHostMdmProfileWithAddedStatus,
  canResendProfiles: boolean,
  canRotateRecoveryLockPassword?: boolean,
  canResendHostNameTemplate?: boolean
) => {
  const { platform, profile_uuid: profileUUID } = row;

  const isAppleMobileConfigProfile =
    isAppleDevice(platform) && !isDDMProfile(row);
  const isWindowsProfile = platform === "windows";
  const isAndroidCertificate =
    platform === "android" &&
    profileUUID === FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID;

  return {
    canResendProfiles:
      canResendProfiles &&
      !SYNTHETIC_PROFILE_UUIDS.includes(profileUUID) &&
      (isWindowsProfile || isAppleMobileConfigProfile || isAndroidCertificate),
    canRotateRecoveryLockPassword:
      profileUUID === REC_LOCK_SYNTHETIC_PROFILE_UUID &&
      canRotateRecoveryLockPassword,
    canResendHostNameTemplate:
      profileUUID === HOST_NAME_SYNTHETIC_PROFILE_UUID &&
      canResendHostNameTemplate,
  };
};

const generateTableConfig = (
  canResendProfiles: boolean,
  resendRequest: (profileUUID: string) => Promise<void>,
  onProfileResent: () => void,
  resendCertificateRequest?: (certificateTemplateId: number) => Promise<void>,
  canRotateRecoveryLockPassword?: boolean,
  rotateRecoveryLockPassword?: () => Promise<void>,
  canResendHostNameTemplate?: boolean,
  resendHostNameTemplate?: () => Promise<void>
): ITableColumnConfig[] => {
  return [
    {
      Header: "Name",
      accessor: "name",
      sortType: "caseInsensitive",
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
      accessor: "status",
      sortType: (
        a: Row<IHostMdmProfileWithAddedStatus>,
        b: Row<IHostMdmProfileWithAddedStatus>
      ) => getStatusSortRank(a.original) - getStatusSortRank(b.original),
      Cell: (cellProps: ITableStringCellProps) => {
        return <OSSettingStatusCell profile={cellProps.row.original} />;
      },
    },
    {
      Header: "Details",
      id: "details",
      accessor: "detail",
      disableSortBy: true,
      // Truncation lives on the cell in _styles.scss — the row itself opens the
      // details modal, so there's no tooltip to reveal the rest.
      Cell: (cellProps: ITableStringCellProps) => {
        return <TextCell value={cellProps.cell.value} className="" />;
      },
    },
    {
      Header: <span className="sr-only">Actions</span>,
      id: "actions",
      disableSortBy: true,
      Cell: (cellProps: ITableStringCellProps) => {
        return (
          <OSSettingsResendCell
            {...getRowActionProps(
              cellProps.row.original,
              canResendProfiles,
              canRotateRecoveryLockPassword,
              canResendHostNameTemplate
            )}
            profile={cellProps.row.original}
            resendRequest={resendRequest}
            resendCertificateRequest={resendCertificateRequest}
            rotateRecoveryLockPassword={rotateRecoveryLockPassword}
            resendHostNameTemplate={resendHostNameTemplate}
            onProfileResent={onProfileResent}
            revealOnRowHover
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
  apple_settings,
  os_settings,
  enrollment_status,
}: IHostMdmData) => {
  let rows: IHostMdmProfileWithAddedStatus[] = profiles ?? [];

  if (apple_settings?.disk_encryption === "action_required") {
    const dERow = profiles?.find(
      (p) => p.name === FLEET_FILEVAULT_PROFILE_DISPLAY_NAME
    );
    if (dERow) {
      // a reference to the original object in rows, so successfully updates it
      dERow.status = "action_required";
    }
  }

  if (
    isEnrolledInMdm(enrollment_status) &&
    os_settings?.recovery_lock_password?.status
  ) {
    rows = [
      ...rows,
      generateRecoveryLockPasswordSetting(
        os_settings.recovery_lock_password.status,
        os_settings.recovery_lock_password.detail
      ),
    ];
  }

  const hostNameRow = generateHostNameSettingIfEligible(
    "darwin",
    enrollment_status,
    os_settings
  );
  if (hostNameRow) {
    rows = [...rows, hostNameRow];
  }

  return rows;
};

// iOS/iPadOS hosts don't surface disk-encryption or recovery-lock rows, but they
// do get the synthetic "Host name" row when a template is enforced. They can also
// have regular configuration profiles.
const makeAppleMobileRows = (
  { profiles, os_settings, enrollment_status }: IHostMdmData,
  platform: ProfilePlatform
) => {
  const rows: IHostMdmProfileWithAddedStatus[] = profiles ? [...profiles] : [];

  const hostNameRow = generateHostNameSettingIfEligible(
    platform,
    enrollment_status,
    os_settings
  );
  if (hostNameRow) {
    rows.push(hostNameRow);
  }

  if (rows.length === 0 && !profiles) {
    return null;
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
    case "zorin":
      return makeLinuxRows(hostMDMData);
    case "rhel":
      return makeLinuxRows(hostMDMData);
    case "ios":
    case "ipados":
      return makeAppleMobileRows(hostMDMData, platform);
    case "android":
      return hostMDMData.profiles;
    default:
      return null;
  }
};

/** Number of controls on the host that are in a failed state. Drives the
 * Controls tab's alert badge. */
export const countFailedControls = (
  rows: IHostMdmProfileWithAddedStatus[] | null | undefined
) => rows?.filter((row) => row.status === "failed").length ?? 0;

export default generateTableConfig;
