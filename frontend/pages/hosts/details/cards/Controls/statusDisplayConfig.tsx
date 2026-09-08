import React from "react";

import {
  FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
  FLEET_FILEVAULT_PROFILE_DISPLAY_NAME,
  HostNameSettingStatus,
  LinuxDiskEncryptionStatus,
  ProfileOperationType,
  RecoveryLockPasswordStatus,
} from "interfaces/mdm";
import { isDDMProfile } from "services/entities/mdm";

import { IconNames } from "components/icons";

import {
  HOST_NAME_SYNTHETIC_PROFILE_UUID,
  REC_LOCK_SYNTHETIC_PROFILE_UUID,
} from "../../helpers";
import {
  IHostMdmProfileWithAddedStatus,
  OsSettingsTableStatusValue,
} from "./OSSettingsTableConfig";

export const isDiskEncryptionProfile = (profileName: string) => {
  return profileName === FLEET_FILEVAULT_PROFILE_DISPLAY_NAME;
};

/** Values interpolated into a control's details message. */
export interface IControlMessageProps {
  /** Display name of the host the control targets. */
  hostDisplayName: string;
  /** Display name of the control, e.g. "Okta Verify settings". */
  settingName: string;
  isDiskEncryptionProfile: boolean;
  /** True on the My device page, where the end user reads the message. */
  isDeviceUser: boolean;
  /** Fleet setting for macOS: disk encryption enforced without key escrow.
   * Disk encryption messages drop their key phrasing. */
  isMacOSDiskEncryptionEnforceOnly?: boolean;
  /** UUID of the profile associated with the control. Can be used with prefix checks to determine the type of profile. */
  profileUUID: string;
}

export type ControlMessage =
  | string
  | ((props: IControlMessageProps) => React.ReactNode);

export type ProfileDisplayOption = {
  statusText: string;
  iconName: IconNames;
  /** Body copy for the control's details modal. */
  message: ControlMessage | null;
} | null;

const actionRequiredMessage: ControlMessage = ({
  isDeviceUser,
  settingName,
}) => {
  const instructions = settingName ? (
    <>
      <b>{settingName}</b> instructions
    </>
  ) : (
    <>instructions</>
  );

  if (isDeviceUser) {
    return (
      <>
        Follow the {instructions} on your <b>My device</b> page.
      </>
    );
  }

  return (
    <>
      Ask the end user to follow the {instructions} on their <b>My device</b>{" "}
      page.
    </>
  );
};

type MacProfileSpecificStatus = "success" | "acknowledged";
type AndroidCertSpecificStatus = "delivered" | "delivering";

export type ProfileStatus = Exclude<
  OsSettingsTableStatusValue,
  AndroidCertSpecificStatus
>;
type OperationTypeOption = Record<ProfileStatus, ProfileDisplayOption>;

type ProfileDisplayConfig = Record<ProfileOperationType, OperationTypeOption>;

const diskEncryptionVerifiedMessage: ControlMessage = ({
  hostDisplayName,
  isMacOSDiskEncryptionEnforceOnly,
}) => (
  <>
    <b>{hostDisplayName}</b> turned disk encryption on
    {!isMacOSDiskEncryptionEnforceOnly && " and sent the key to Fleet"}. Fleet
    verified.
  </>
);

const diskEncryptionVerifyingMessage: ControlMessage = ({
  hostDisplayName,
  isMacOSDiskEncryptionEnforceOnly,
}) => (
  <>
    <b>{hostDisplayName}</b> acknowledged the MDM command to turn on disk
    encryption. Fleet is verifying with osquery
    {!isMacOSDiskEncryptionEnforceOnly &&
      " and retrieving the disk encryption key"}
    . This may take up to one hour.
  </>
);

const diskEncryptionEnforcingMessage: ControlMessage = ({
  hostDisplayName,
}) => (
  <>
    <b>{hostDisplayName}</b> will receive the MDM command to turn on disk
    encryption when the host comes online.
  </>
);

const diskEncryptionFailedMessage: ControlMessage = ({ hostDisplayName }) => (
  <>
    <b>{hostDisplayName}</b> failed to turn on disk encryption.
  </>
);

// Profiles for iOS and iPadOS skip the verifying step
const APPLE_PROFILE_VERIFIED_DISPLAY_CONFIG: ProfileDisplayOption = {
  statusText: "Verified",
  iconName: "success",
  message: (props) =>
    props.isDiskEncryptionProfile ? (
      diskEncryptionVerifiedMessage(props)
    ) : (
      <>
        <b>{props.hostDisplayName}</b> applied <b>{props.settingName}</b>. Fleet
        verified.
      </>
    ),
};

const MAC_PROFILE_VERIFYING_DISPLAY_CONFIG: ProfileDisplayOption = {
  statusText: "Verifying",
  iconName: "success-outline",
  message: (props) =>
    props.isDiskEncryptionProfile ? (
      diskEncryptionVerifyingMessage(props)
    ) : (
      <>
        <b>{props.hostDisplayName}</b> acknowledged the MDM command to apply{" "}
        <b>{props.settingName}</b>.
        {!isDDMProfile({ profile_uuid: props.profileUUID }) && (
          <> Fleet is verifying with osquery.</>
        )}
      </>
    ),
};

export const PROFILE_DISPLAY_CONFIG: ProfileDisplayConfig = {
  install: {
    verified: APPLE_PROFILE_VERIFIED_DISPLAY_CONFIG,
    success: APPLE_PROFILE_VERIFIED_DISPLAY_CONFIG,
    verifying: MAC_PROFILE_VERIFYING_DISPLAY_CONFIG,
    acknowledged: MAC_PROFILE_VERIFYING_DISPLAY_CONFIG,
    pending: {
      statusText: "Enforcing",
      iconName: "pending-outline",
      message: (props) =>
        props.isDiskEncryptionProfile ? (
          diskEncryptionEnforcingMessage(props)
        ) : (
          <>
            <b>{props.hostDisplayName}</b> is running the MDM command to apply{" "}
            <b>{props.settingName}</b> or will run it when the host comes
            online.
          </>
        ),
    },
    action_required: {
      statusText: "Action required",
      iconName: "pending-outline",
      message: actionRequiredMessage,
    },
    failed: {
      statusText: "Failed",
      iconName: "error",
      message: (props) =>
        props.isDiskEncryptionProfile ? (
          diskEncryptionFailedMessage(props)
        ) : (
          <>
            <b>{props.hostDisplayName}</b> failed to apply{" "}
            <b>{props.settingName}</b>.
          </>
        ),
    },
  },
  remove: {
    pending: {
      statusText: "Removing enforcement",
      iconName: "pending-outline",
      message: (props) =>
        props.isDiskEncryptionProfile ? (
          <>
            <b>{props.hostDisplayName}</b> will receive the MDM command to
            remove the disk encryption profile when the host comes online.
          </>
        ) : (
          <>
            <b>{props.hostDisplayName}</b> is running the MDM command to remove{" "}
            <b>{props.settingName}</b> or will run it when the host comes
            online.
          </>
        ),
    },
    action_required: null, // should not be reached
    verified: null, // should not be reached
    verifying: null, // should not be reached
    success: null, // should not be reached
    acknowledged: null, // should not be reached
    failed: {
      statusText: "Failed",
      iconName: "error",
      message: (props) =>
        props.isDiskEncryptionProfile ? (
          <>
            <b>{props.hostDisplayName}</b> failed to remove the disk encryption
            profile.
          </>
        ) : (
          <>
            <b>{props.hostDisplayName}</b> failed to remove{" "}
            <b>{props.settingName}</b>.
          </>
        ),
    },
  },
};

export type WindowsDiskEncryptionDisplayStatus = Exclude<
  ProfileStatus,
  MacProfileSpecificStatus | AndroidCertSpecificStatus
>;

type WindowsDiskEncryptionDisplayConfig = Pick<
  OperationTypeOption,
  WindowsDiskEncryptionDisplayStatus
>;

export const WINDOWS_DISK_ENCRYPTION_DISPLAY_CONFIG: WindowsDiskEncryptionDisplayConfig = {
  verified: {
    statusText: "Verified",
    iconName: "success",
    message: diskEncryptionVerifiedMessage,
  },
  verifying: {
    statusText: "Verifying",
    iconName: "success-outline",
    message: diskEncryptionVerifyingMessage,
  },
  pending: {
    statusText: "Enforcing",
    iconName: "pending-outline",
    message: diskEncryptionEnforcingMessage,
  },
  action_required: {
    statusText: "Action required",
    iconName: "pending-outline",
    // Defensive fallback only. The server always sends a reason for this status and the details modal prefers it, so
    // this renders only if that reason is ever missing.
    message: ({ hostDisplayName }) => (
      <>
        Disk encryption on <b>{hostDisplayName}</b> needs attention.
      </>
    ),
  },
  failed: {
    statusText: "Failed",
    iconName: "error",
    message: diskEncryptionFailedMessage,
  },
};

type LinuxDiskEncryptionDisplayConfig = Omit<
  OperationTypeOption,
  MacProfileSpecificStatus | AndroidCertSpecificStatus | "pending" | "verifying"
>;

export const LINUX_DISK_ENCRYPTION_DISPLAY_CONFIG: LinuxDiskEncryptionDisplayConfig = {
  verified: {
    statusText: "Verified",
    iconName: "success",
    message: diskEncryptionVerifiedMessage,
  },
  failed: {
    statusText: "Failed",
    iconName: "error",
    message: diskEncryptionFailedMessage,
  },
  action_required: {
    statusText: "Action required",
    iconName: "pending-outline",
    message: actionRequiredMessage,
  },
};

export const HOST_NAME_DISPLAY_CONFIG: Record<
  HostNameSettingStatus,
  ProfileDisplayOption
> = {
  pending: {
    statusText: "Enforcing",
    iconName: "pending-outline",
    message: ({ hostDisplayName }) => (
      <>
        <b>{hostDisplayName}</b> will be renamed to match this fleet&apos;s host
        name template when it comes online.
      </>
    ),
  },
  verifying: {
    statusText: "Verifying",
    iconName: "success-outline",
    message: ({ hostDisplayName }) => (
      <>
        <b>{hostDisplayName}</b> acknowledged the MDM command to rename it.
        Fleet is verifying.
      </>
    ),
  },
  verified: {
    statusText: "Verified",
    iconName: "success",
    message: ({ hostDisplayName }) => (
      <>
        <b>{hostDisplayName}</b> was renamed to match this fleet&apos;s host
        name template. Fleet verified.
      </>
    ),
  },
  failed: {
    statusText: "Failed",
    iconName: "error",
    message: ({ hostDisplayName }) => (
      <>
        <b>{hostDisplayName}</b> failed to apply this fleet&apos;s host name
        template.
      </>
    ),
  },
};

export const RECOVERY_LOCK_PASSWORD_DISPLAY_CONFIG: Record<
  RecoveryLockPasswordStatus,
  ProfileDisplayOption
> = {
  verified: {
    statusText: "Verified",
    iconName: "success",
    message: ({ hostDisplayName }) => (
      <>
        Fleet set a recovery lock password for <b>{hostDisplayName}</b>.
      </>
    ),
  },
  pending: {
    statusText: "Enforcing",
    iconName: "pending-outline",
    message: ({ hostDisplayName }) => (
      <>
        Fleet is setting a recovery lock password for <b>{hostDisplayName}</b>.
      </>
    ),
  },
  removing_enforcement: {
    statusText: "Removing enforcement",
    iconName: "pending-outline",
    message: ({ hostDisplayName }) => (
      <>
        Fleet is unsetting the recovery lock password for{" "}
        <b>{hostDisplayName}</b>.
      </>
    ),
  },
  failed: {
    statusText: "Failed",
    iconName: "error",
    message: ({ hostDisplayName }) => (
      <>
        Fleet failed to set a recovery lock password for{" "}
        <b>{hostDisplayName}</b>.
      </>
    ),
  },
};

const getAndroidCertificateDisplayOption = (
  status: OsSettingsTableStatusValue,
  operationType: ProfileOperationType | null
): ProfileDisplayOption => {
  switch (status) {
    case "pending":
    case "delivering":
    case "delivered":
      return operationType === "install"
        ? {
            statusText: "Enforcing",
            iconName: "pending-outline",
            // Android goes over the Management API, not an MDM command.
            message: ({ hostDisplayName, settingName }) => (
              <>
                <b>{hostDisplayName}</b> is running the command to apply{" "}
                <b>{settingName}</b> or will run it when the host comes online.
              </>
            ),
          }
        : {
            statusText: "Removing enforcement",
            iconName: "pending-outline",
            message: ({ hostDisplayName, settingName }) => (
              <>
                <b>{hostDisplayName}</b> is running the command to remove{" "}
                <b>{settingName}</b> or will run it when the host comes online.
              </>
            ),
          };
    case "verified":
      return {
        statusText: "Verified",
        iconName: "success",
        message: ({ hostDisplayName, settingName }) => (
          <>
            <b>{hostDisplayName}</b> applied <b>{settingName}</b>. Fleet
            verified.
          </>
        ),
      };
    case "failed":
      return {
        statusText: "Failed",
        iconName: "error",
        message: ({ hostDisplayName, settingName }) => (
          <>
            <b>{hostDisplayName}</b> failed to apply <b>{settingName}</b>.
          </>
        ),
      };
    default:
      return null;
  }
};

/** Icon, status text, and details message for a control row. Shared by the
 * table's status cell and the details modal so the two can't drift. */
export const getControlDisplayOption = (
  row: IHostMdmProfileWithAddedStatus
): ProfileDisplayOption => {
  const { status, operation_type: operationType, platform } = row;
  const profileUUID = row.profile_uuid;

  if (platform === "linux") {
    return LINUX_DISK_ENCRYPTION_DISPLAY_CONFIG[
      status as LinuxDiskEncryptionStatus
    ];
  }

  if (profileUUID === REC_LOCK_SYNTHETIC_PROFILE_UUID) {
    return RECOVERY_LOCK_PASSWORD_DISPLAY_CONFIG[
      status as RecoveryLockPasswordStatus
    ];
  }

  if (profileUUID === HOST_NAME_SYNTHETIC_PROFILE_UUID) {
    return HOST_NAME_DISPLAY_CONFIG[status as HostNameSettingStatus];
  }

  if (
    platform === "android" &&
    profileUUID === FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID
  ) {
    return getAndroidCertificateDisplayOption(status, operationType);
  }

  // The synthesized Windows disk encryption row has no operation type.
  if (!operationType && status !== "success" && status !== "acknowledged") {
    return WINDOWS_DISK_ENCRYPTION_DISPLAY_CONFIG[
      status as WindowsDiskEncryptionDisplayStatus
    ];
  }

  if (operationType) {
    return PROFILE_DISPLAY_CONFIG[operationType]?.[status as ProfileStatus];
  }

  return null;
};
