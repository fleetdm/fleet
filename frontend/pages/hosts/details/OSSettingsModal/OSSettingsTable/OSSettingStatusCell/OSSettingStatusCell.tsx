import React from "react";
import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import {
  FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID,
  LinuxDiskEncryptionStatus,
  ProfileOperationType,
  ProfilePlatform,
} from "interfaces/mdm";
import { COLORS } from "styles/var/colors";

import { OsSettingsTableStatusValue } from "../OSSettingsTableConfig";
import TooltipContent from "./components/Tooltip/TooltipContent";
import {
  isDiskEncryptionProfile,
  LINUX_DISK_ENCRYPTION_DISPLAY_CONFIG,
  PROFILE_DISPLAY_CONFIG,
  ProfileDisplayOption,
  ProfileStatus,
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
}

const OSSettingStatusCell = ({
  status,
  operationType,
  profileName = "",
  hostPlatform,
  profileUUID,
}: IOSSettingStatusCellProps) => {
  let displayOption: ProfileDisplayOption = null;
  if (hostPlatform === "linux") {
    displayOption =
      LINUX_DISK_ENCRYPTION_DISPLAY_CONFIG[status as LinuxDiskEncryptionStatus];
  }

  // Android host certificate templates.
  else if (
    hostPlatform === "android" &&
    profileUUID === FLEET_ANDROID_CERTIFICATE_TEMPLATE_PROFILE_ID
  ) {
    switch (status) {
      case "pending":
        if (operationType === "install") {
          displayOption = {
            statusText: "Enforcing (pending)",
            iconName: "pending-outline",
            tooltip:
              "The host is running the command to apply settings or will run it when the host comes online.",
          };
        } else {
          displayOption = {
            statusText: "Removing enforcement (pending)",
            iconName: "pending-outline",
            tooltip:
              "The host is running the command to remove settings or will run it when the host comes online.",
          };
        }
        break;
      case "delivering":
      case "delivered":
        if (operationType === "install") {
          // note that thise case is identical to the "pending" case above for install operation
          // separation allows catching the below error case
          displayOption = {
            statusText: "Enforcing (pending)",
            iconName: "pending-outline",
            tooltip:
              "The host is running the command to apply settings or will run it when the host comes online.",
          };
        } else {
          throw new Error(
            "Received unexpected 'delivering' or 'delivered' status for remove operation"
          );
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
    const tooltipId = uniqueId();
    return (
      <span className={baseClass}>
        <Icon name={iconName} />
        {tooltip ? (
          <>
            <span
              className={`${baseClass}__status-text`}
              data-tip
              data-for={tooltipId}
              data-tip-disable={false}
            >
              {statusText}
            </span>
            <ReactTooltip
              place="top"
              effect="solid"
              backgroundColor={COLORS["tooltip-bg"]}
              id={tooltipId}
              data-html
            >
              <span className="tooltip__tooltip-text">
                {status !== "action_required" ? (
                  <TooltipContent
                    innerContent={tooltip}
                    innerProps={{
                      isDiskEncryptionProfile: isDiskEncryptionProfile(
                        profileName
                      ),
                    }}
                  />
                ) : (
                  <TooltipContent
                    innerContent={tooltip}
                    innerProps={{ isDeviceUser, profileName }}
                  />
                )}
              </span>
            </ReactTooltip>
          </>
        ) : (
          statusText
        )}
      </span>
    );
  }
  // graceful error - this state should not be reached based on the API spec
  return <TextCell value="Unrecognized" />;
};
export default OSSettingStatusCell;
