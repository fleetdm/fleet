import { ProfileOperationType } from "interfaces/mdm";

import { IconNames } from "components/icons";
import {
  TooltipInnerContentFunc,
  TooltipInnerContentOption,
} from "./components/Tooltip/TooltipContent";

import { OsSettingsTableStatusValue } from "../OSSettingsTableConfig";
import TooltipInnerContentActionRequired from "./components/Tooltip/ActionRequired";

export type ProfileDisplayOption = {
  statusText: string;
  iconName: IconNames;
  tooltip: TooltipInnerContentOption | null;
} | null;

type OperationTypeOption = Record<
  OsSettingsTableStatusValue,
  ProfileDisplayOption
>;

type ProfileDisplayConfig = Record<ProfileOperationType, OperationTypeOption>;

const MAC_PROFILE_VERIFIED_DISPLAY_CONFIG: ProfileDisplayOption = {
  statusText: "Verified",
  iconName: "success",
  tooltip: (innerProps) =>
    innerProps.isDiskEncryptionProfile
      ? "The host turned disk encryption on and sent the key to Fleet. " +
        "Fleet verified with osquery."
      : "The host applied the setting. Fleet verified with osquery. " +
        "Declaration profiles are verified with DDM.",
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
    verified: MAC_PROFILE_VERIFIED_DISPLAY_CONFIG,
    success: MAC_PROFILE_VERIFIED_DISPLAY_CONFIG,
    verifying: MAC_PROFILE_VERIFYING_DISPLAY_CONFIG,
    acknowledged: MAC_PROFILE_VERIFYING_DISPLAY_CONFIG,
    pending: {
      statusText: "Enforcing (pending)",
      iconName: "pending-outline",
      tooltip: (innerProps) =>
        innerProps.isDiskEncryptionProfile
          ? "The hosts will receive the MDM command to turn on disk encryption " +
            "when the hosts come online."
          : "The host will receive the MDM command to apply the setting when the " +
            "host comes online.",
    },
    action_required: {
      statusText: "Action required (pending)",
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
      statusText: "Removing enforcement (pending)",
      iconName: "pending-outline",
      tooltip: (innerProps) =>
        innerProps.isDiskEncryptionProfile
          ? "The host will receive the MDM command to remove the disk encryption profile when the " +
            "host comes online."
          : "The host will receive the MDM command to remove the setting when the host " +
            "comes online.",
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

type WindowsDiskEncryptionDisplayConfig = Omit<
  OperationTypeOption,
  // windows disk encryption does not have these states
  "action_required" | "success" | "acknowledged"
>;

export const WINDOWS_DISK_ENCRYPTION_DISPLAY_CONFIG: WindowsDiskEncryptionDisplayConfig = {
  verified: {
    statusText: "Verified",
    iconName: "success",
    tooltip: () =>
      "The host turned disk encryption on and sent the key to Fleet. Fleet verified with osquery.",
  },
  verifying: {
    statusText: "Verifying",
    iconName: "success-outline",
    tooltip: () =>
      "The host acknowledged the MDM command to turn on disk encryption. Fleet is verifying with " +
      "osquery and retrieving the disk encryption key. This may take up to one hour.",
  },
  pending: {
    statusText: "Enforcing (pending)",
    iconName: "pending-outline",
    tooltip: () =>
      "The host will receive the MDM command to turn on disk encryption when the host comes online.",
  },
  failed: {
    statusText: "Failed",
    iconName: "error",
    tooltip: null,
  },
};
