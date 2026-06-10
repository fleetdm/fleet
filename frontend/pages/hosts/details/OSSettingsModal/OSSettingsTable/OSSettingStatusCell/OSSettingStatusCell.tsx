import React from "react";

import { REC_LOCK_SYNTHETIC_PROFILE_UUID } from "pages/hosts/details/helpers";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import {
  FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
  LinuxDiskEncryptionStatus,
  ProfileOperationType,
  ProfilePlatform,
  RecoveryLockPasswordStatus,
} from "interfaces/mdm";
import TooltipWrapper from "components/TooltipWrapper";

import {
  IHostMdmProfileWithAddedStatus,
  OsSettingsTableStatusValue,
} from "../OSSettingsTableConfig";
import TooltipContent from "./components/Tooltip/TooltipContent";
import generateErrorTooltip from "./errorTooltipHelpers";
import {
  isDiskEncryptionProfile,
  LINUX_DISK_ENCRYPTION_DISPLAY_CONFIG,
  PROFILE_DISPLAY_CONFIG,
  ProfileDisplayOption,
  ProfileStatus,
  RECOVERY_LOCK_PASSWORD_DISPLAY_CONFIG,
  WINDOWS_DISK_ENCRYPTION_DISPLAY_CONFIG,
  WindowsDiskEncryptionDisplayStatus,
} from "./helpers";

const baseClass = "os-settings-status-cell";

interface IOSSettingStatusCellProps {
  status: OsSettingsTableStatusValue;
  operationType: ProfileOperationType | null;
  profileName: string;
  hostPlatform?: ProfilePlatform;
  profileUUID?: string;
  profile?: IHostMdmProfileWithAddedStatus;
}

const OSSettingStatusCell = ({
  status,
  operationType,
  profileName = "",
  hostPlatform,
  profileUUID,
  profile,
}: IOSSettingStatusCellProps) => {
  let displayOption: ProfileDisplayOption = null;
  if (hostPlatform === "linux") {
    displayOption =
      LINUX_DISK_ENCRYPTION_DISPLAY_CONFIG[status as LinuxDiskEncryptionStatus];
  } else if (profileUUID === REC_LOCK_SYNTHETIC_PROFILE_UUID) {
    displayOption =
      RECOVERY_LOCK_PASSWORD_DISPLAY_CONFIG[
        status as RecoveryLockPasswordStatus
      ];
  }

  // Android host certificate templates.
  else if (
    hostPlatform === "android" &&
    profileUUID === FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID
  ) {
    switch (status) {
      case "pending":
      case "delivering":
      case "delivered":
        if (operationType === "install") {
          displayOption = {
            statusText: "Enforcing",
            iconName: "pending-outline",
            tooltip:
              "The host is running the command to apply settings or will run it when the host comes online.",
          };
        } else {
          displayOption = {
            statusText: "Removing enforcement",
            iconName: "pending-outline",
            tooltip:
              "The host is running the command to remove settings or will run it when the host comes online.",
          };
        }
        break;
      case "verified":
        displayOption = {
          statusText: "Verified",
          iconName: "success",
          tooltip: () => "The host applied the setting. Fleet verified",
        };
        break;
      case "failed":
        displayOption = {
          statusText: "Failed",
          iconName: "error",
          tooltip: null,
        };
        break;
      default:
        displayOption = null;
    }
  }

  // windows hosts do not have an operation type at the moment and their display options are
  // different than mac hosts.
  else if (
    !operationType &&
    status !== "success" &&
    status !== "acknowledged"
  ) {
    displayOption =
      WINDOWS_DISK_ENCRYPTION_DISPLAY_CONFIG[
        status as WindowsDiskEncryptionDisplayStatus
      ];
  } else if (operationType) {
    displayOption =
      PROFILE_DISPLAY_CONFIG[operationType]?.[status as ProfileStatus];
  }

  const isDeviceUser = window.location.pathname
    .toLowerCase()
    .includes("/device/");

  if (displayOption) {
    const { statusText, iconName, tooltip } = displayOption;

    // For failed status, use the error detail as tooltip content
    const errorTooltip = profile ? generateErrorTooltip(profile) : null;
    // For pending profiles, prefer a backend-provided detail message (e.g.
    // Android Wi-Fi profiles waiting for their certificate) over the generic
    // "Enforcing" tooltip.
    const pendingDetailTooltip =
      profile?.status === "pending" && profile.detail ? profile.detail : null;

    let tipContent: React.ReactNode;
    if (pendingDetailTooltip) {
      tipContent = (
        <span className="tooltip__tooltip-text">{pendingDetailTooltip}</span>
      );
    } else if (tooltip) {
      if (status !== "action_required") {
        tipContent = (
          <span className="tooltip__tooltip-text">
            <TooltipContent
              innerContent={tooltip}
              innerProps={{
                isDiskEncryptionProfile: isDiskEncryptionProfile(profileName),
              }}
            />
          </span>
        );
      } else {
        tipContent = (
          <span className="tooltip__tooltip-text">
            <TooltipContent
              innerContent={tooltip}
              innerProps={{ isDeviceUser, profileName }}
            />
          </span>
        );
      }
    } else if (errorTooltip) {
      tipContent = (
        <span className="tooltip__tooltip-text">{errorTooltip}</span>
      );
    }

    return (
      <span className={baseClass}>
        <Icon name={iconName} />
        {tipContent ? (
          <TooltipWrapper
            tipContent={tipContent}
            position="top"
            underline={false}
            showArrow
            tipOffset={8}
            clickable
          >
            <span className={`${baseClass}__status-text`}>{statusText}</span>
          </TooltipWrapper>
        ) : (
          <span className={`${baseClass}__status-text`}>{statusText}</span>
        )}
      </span>
    );
  }
  // graceful error - this state should not be reached based on the API spec
  return <TextCell value="Unrecognized" />;
};
export default OSSettingStatusCell;
