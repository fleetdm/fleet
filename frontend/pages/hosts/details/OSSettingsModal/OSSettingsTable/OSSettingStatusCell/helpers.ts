import {
  FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
  FLEET_FILEVAULT_PROFILE_DISPLAY_NAME,
  HostNameSettingStatus,
  ProfileOperationType,
  RecoveryLockPasswordStatus,
} from "interfaces/mdm";

import { IconNames } from "components/icons";
import {
  TooltipInnerContentFunc,
  TooltipInnerContentOption,
} from "./components/Tooltip/TooltipContent";

import {
  IHostMdmProfileWithAddedStatus,
  OsSettingsTableStatusValue,
} from "../OSSettingsTableConfig";
import TooltipInnerContentActionRequired from "./components/Tooltip/ActionRequired";
import { formatAndroidCertificateError } from "./errorTooltipHelpers";

export const isDiskEncryptionProfile = (profileName: string) => {
  return profileName === FLEET_FILEVAULT_PROFILE_DISPLAY_NAME;
};

export type ProfileDisplayOption = {
  statusText: string;
  iconName: IconNames;
  tooltip: TooltipInnerContentOption | null;
} | null;

/**
 * When an Android certificate install fails, Fleet resets the certificate and delivers it again,
 * so the profile goes back to an in-progress status and is otherwise indistinguishable from a
 * first delivery. The server decides this: a manual resend leaves the same retry count behind, and
 * telling the two apart needs state that does not survive into the API response.
 */
export const isRetryingAndroidCertificate = (
  profile?: IHostMdmProfileWithAddedStatus
) =>
  profile?.profile_uuid === FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID &&
  !!profile.retrying;

/**
 * Builds the tooltip for a retrying Android certificate, e.g. "Network error during SCEP
 * enrollment. Retrying enrollment (attempt 2 of 4)." The numbers are labelled as attempts because
 * they count the initial attempt, and so run one higher than the retry_count/max_retries the API
 * reports. The host does not have to report an error message with a failure, in which case the
 * retry stands on its own.
 */
export const getAndroidCertificateRetryTooltip = (
  profile: IHostMdmProfileWithAddedStatus
) => {
  const attempts =
    profile.retry_count === undefined || profile.max_retries === undefined
      ? ""
      : ` (attempt ${profile.retry_count + 1} of ${profile.max_retries + 1})`;
  const retrying = `Retrying enrollment${attempts}.`;

  const reported = profile.detail?.trim();
  if (!reported) {
    return retrying;
  }

  // Prefer plain language over the app's own wording, falling back to what it reported so an
  // unrecognized failure still reaches the reader.
  const detail = formatAndroidCertificateError(reported) ?? reported;
  const sentence = /[.!?]$/.test(detail) ? detail : `${detail}.`;
  return `${sentence} ${retrying}`;
};

export const ANDROID_CERT_RETRYING_DISPLAY_CONFIG: ProfileDisplayOption = {
  statusText: "Retrying",
  // Deliberately the same icon the in-progress statuses use, rather than a warning icon: a retry
  // is still in flight, and a new status icon would have to be introduced across the rest of the
  // OS settings UI. The status text and tooltip carry the difference.
  iconName: "pending-outline",
  // The tooltip is built from the profile's detail, see getAndroidCertificateRetryTooltip.
  tooltip: null,
};

type MacProfileSpecificStatus = "success" | "acknowledged";
type AndroidCertSpecificStatus = "delivered" | "delivering";

export type ProfileStatus = Exclude<
  OsSettingsTableStatusValue,
  AndroidCertSpecificStatus
>;
type OperationTypeOption = Record<ProfileStatus, ProfileDisplayOption>;

type ProfileDisplayConfig = Record<ProfileOperationType, OperationTypeOption>;

// Profiles for iOS and iPadOS skip the verifying step
const APPLE_PROFILE_VERIFIED_DISPLAY_CONFIG: ProfileDisplayOption = {
  statusText: "Verified",
  iconName: "success",
  tooltip: (innerProps) =>
    innerProps.isDiskEncryptionProfile
      ? "The host turned disk encryption on and sent the key to Fleet. " +
        "Fleet verified."
      : "The host applied the setting. Fleet verified.",
} as const;

const MAC_PROFILE_VERIFYING_DISPLAY_CONFIG: ProfileDisplayOption = {
  statusText: "Verifying",
  iconName: "success-outline",
  tooltip: (innerProps) =>
    innerProps.isDiskEncryptionProfile
      ? "The host acknowledged the MDM command to turn on disk encryption. " +
        "Fleet is verifying with osquery and retrieving the disk encryption key. " +
        "This may take up to one hour."
      : "The host acknowledged the MDM command to apply the setting. Fleet is " +
        "verifying with osquery.",
} as const;

export const PROFILE_DISPLAY_CONFIG: ProfileDisplayConfig = {
  install: {
    verified: APPLE_PROFILE_VERIFIED_DISPLAY_CONFIG,
    success: APPLE_PROFILE_VERIFIED_DISPLAY_CONFIG,
    verifying: MAC_PROFILE_VERIFYING_DISPLAY_CONFIG,
    acknowledged: MAC_PROFILE_VERIFYING_DISPLAY_CONFIG,
    pending: {
      statusText: "Enforcing",
      iconName: "pending-outline",
      tooltip: (innerProps) =>
        innerProps.isDiskEncryptionProfile
          ? "The hosts will receive the MDM command to turn on disk encryption " +
            "when the hosts come online."
          : "The host is running the MDM command to apply settings or will run it " +
            "when the host comes online.",
    },
    action_required: {
      statusText: "Action required",
      iconName: "pending-outline",
      tooltip: TooltipInnerContentActionRequired as TooltipInnerContentFunc,
    },
    failed: {
      statusText: "Failed",
      iconName: "error",
      tooltip: null,
    },
  },
  remove: {
    pending: {
      statusText: "Removing enforcement",
      iconName: "pending-outline",
      tooltip: (innerProps) =>
        innerProps.isDiskEncryptionProfile
          ? "The host will receive the MDM command to remove the disk encryption profile when the " +
            "host comes online."
          : "The host is running the MDM command to remove settings or will run " +
            "it when the host comes online.",
    },
    action_required: null, // should not be reached
    verified: null, // should not be reached
    verifying: null, // should not be reached
    success: null, // should not be reached
    acknowledged: null, // should not be reached
    failed: {
      statusText: "Failed",
      iconName: "error",
      tooltip: null,
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
    tooltip: () =>
      "The host turned disk encryption on and sent the key to Fleet. Fleet verified.",
  },
  verifying: {
    statusText: "Verifying",
    iconName: "success-outline",
    tooltip: () =>
      "The host acknowledged the MDM command to turn on disk encryption. Fleet is verifying with " +
      "osquery and retrieving the disk encryption key. This may take up to one hour.",
  },
  pending: {
    statusText: "Enforcing",
    iconName: "pending-outline",
    tooltip: () =>
      "The host will receive the MDM command to turn on disk encryption when the host comes online.",
  },
  action_required: {
    statusText: "Action required",
    iconName: "pending-outline",
    tooltip: () =>
      "Disk encryption is on, but the user has not set a BitLocker PIN yet.",
  },
  failed: {
    statusText: "Failed",
    iconName: "error",
    tooltip: null,
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
    tooltip: () =>
      "The host turned disk encryption on and sent the key to Fleet. Fleet verified.",
  },
  failed: {
    statusText: "Failed",
    iconName: "error",
    tooltip: null,
  },
  action_required: {
    statusText: "Action required",
    iconName: "pending-outline",
    tooltip: TooltipInnerContentActionRequired as TooltipInnerContentFunc,
  },
};

export const HOST_NAME_DISPLAY_CONFIG: Record<
  HostNameSettingStatus,
  ProfileDisplayOption
> = {
  pending: {
    statusText: "Enforcing",
    iconName: "pending-outline",
    tooltip:
      "Fleet is enforcing this fleet's host name template. The host will be renamed when it comes online.",
  },
  verifying: {
    statusText: "Verifying",
    iconName: "success-outline",
    tooltip:
      "The host acknowledged the MDM command to rename it. Fleet is verifying.",
  },
  verified: {
    statusText: "Verified",
    iconName: "success",
    tooltip:
      "The host was renamed to match this fleet's host name template. Fleet verified.",
  },
  // failed has no static tooltip so the cell falls back to the error-detail
  // tooltip (drift message or Apple error) via generateErrorTooltip.
  failed: {
    statusText: "Failed",
    iconName: "error",
    tooltip: null,
  },
};

export const RECOVERY_LOCK_PASSWORD_DISPLAY_CONFIG: Record<
  RecoveryLockPasswordStatus,
  ProfileDisplayOption
> = {
  verified: {
    statusText: "Verified",
    iconName: "success",
    tooltip: "Fleet set a recovery lock password for the host.",
  },
  pending: {
    statusText: "Enforcing",
    iconName: "pending-outline",
    tooltip: "Fleet is setting a recovery lock password for the host.",
  },
  removing_enforcement: {
    statusText: "Removing enforcement",
    iconName: "pending-outline",
    tooltip: "Fleet is unsetting the recovery lock password for the host.",
  },
  failed: {
    statusText: "Failed",
    iconName: "error",
    tooltip: "Fleet failed to set a recovery lock password for the host.",
  },
};
