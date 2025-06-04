import React from "react";
import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";

import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import {
  LinuxDiskEncryptionStatus,
  ProfileOperationType,
  ProfilePlatform,
} from "interfaces/mdm";
import { COLORS } from "styles/var/colors";

import {
  isMdmProfileStatus,
  OsSettingsTableStatusValue,
} from "../OSSettingsTableConfig";
import TooltipContent from "./components/Tooltip/TooltipContent";
import {
  isDiskEncryptionProfile,
  LINUX_DISK_ENCRYPTION_DISPLAY_CONFIG,
  PROFILE_DISPLAY_CONFIG,
  ProfileDisplayOption,
  WINDOWS_DISK_ENCRYPTION_DISPLAY_CONFIG,
} from "./helpers";

const baseClass = "os-settings-status-cell";

interface IOSSettingStatusCellProps {
  status: OsSettingsTableStatusValue;
  operationType: ProfileOperationType | null;
  profileName: string;
  hostPlatform?: ProfilePlatform;
}

const OSSettingStatusCell = ({
  status,
  operationType,
  profileName = "",
  hostPlatform,
}: IOSSettingStatusCellProps) => {
  let displayOption: ProfileDisplayOption = null;

  if (hostPlatform === "linux") {
    displayOption =
      LINUX_DISK_ENCRYPTION_DISPLAY_CONFIG[status as LinuxDiskEncryptionStatus];
  }

  // windows hosts do not have an operation type at the moment and their display options are
  // different than mac hosts.
  else if (!operationType && isMdmProfileStatus(status)) {
    displayOption = WINDOWS_DISK_ENCRYPTION_DISPLAY_CONFIG[status];
  } else if (operationType) {
    displayOption = PROFILE_DISPLAY_CONFIG[operationType]?.[status];
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
