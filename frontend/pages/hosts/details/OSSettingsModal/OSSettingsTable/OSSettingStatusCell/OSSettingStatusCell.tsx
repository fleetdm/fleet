import React from "react";
import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import TextCell from "components/TableContainer/DataTable/TextCell";
import {
  FLEET_FILEVAULT_PROFILE_DISPLAY_NAME,
  MacMdmProfileOperationType,
} from "interfaces/mdm";

import {
  isMdmProfileStatus,
  OsSettingsTableStatusValue,
} from "../OSSettingsTableConfig";
import TooltipContent, {
  TooltipInnerContentFunc,
  TooltipInnerContentOption,
} from "./components/Tooltip/TooltipContent";
import TooltipInnerContentActionRequired from "./components/Tooltip/ActionRequired";

const baseClass = "os-setting-status-cell";

type ProfileDisplayOption = {
  statusText: string;
  iconName: IconNames;
  tooltip: TooltipInnerContentOption | null;
} | null;

type OperationTypeOption = Record<
  OsSettingsTableStatusValue,
  ProfileDisplayOption
>;
type ProfileDisplayConfig = Record<
  MacMdmProfileOperationType,
  OperationTypeOption
>;

const PROFILE_DISPLAY_CONFIG: ProfileDisplayConfig = {
  install: {
    pending: {
      statusText: "Enforcing (pending)",
      iconName: "pending-partial",
      tooltip: (innerProps) =>
        innerProps.isDiskEncryptionProfile
          ? "The hosts will receive the MDM command to turn on disk encryption " +
            "when the hosts come online."
          : "The host will receive the MDM command to install the configuration profile when the " +
            "host comes online.",
    },
    action_required: {
      statusText: "Action required (pending)",
      iconName: "pending-partial",
      tooltip: TooltipInnerContentActionRequired as TooltipInnerContentFunc,
    },
    verified: {
      statusText: "Verified",
      iconName: "success",
      tooltip: (innerProps) =>
        innerProps.isDiskEncryptionProfile
          ? "The host turned disk encryption on and sent the key to Fleet. " +
            "Fleet verified with osquery."
          : "The host installed the configuration profile. Fleet verified with osquery.",
    },
    verifying: {
      statusText: "Verifying",
      iconName: "success-partial",
      tooltip: (innerProps) =>
        innerProps.isDiskEncryptionProfile
          ? "The host acknowledged the MDM command to turn on disk encryption. " +
            "Fleet is verifying with osquery and retrieving the disk encryption key. " +
            "This may take up to one hour."
          : "The host acknowledged the MDM command to install the configuration profile. Fleet is " +
            "verifying with osquery.",
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
      iconName: "pending-partial",
      tooltip: (innerProps) =>
        innerProps.isDiskEncryptionProfile
          ? "The host will receive the MDM command to remove the disk encryption profile when the " +
            "host comes online."
          : "The host will receive the MDM command to remove the configuration profile when the host " +
            "comes online.",
    },
    action_required: null, // should not be reached
    verified: null, // should not be reached
    verifying: null, // should not be reached
    failed: {
      statusText: "Failed",
      iconName: "error",
      tooltip: null,
    },
  },
};

type WindowsDiskEncryptionDisplayConfig = Omit<
  OperationTypeOption,
  "action_required"
>;

const WINDOWS_DISK_ENCRYPTION_DISPLAY_CONFIG: WindowsDiskEncryptionDisplayConfig = {
  verified: {
    statusText: "Verified",
    iconName: "success",
    tooltip: () =>
      "The host turned disk encryption on and sent the key to Fleet. Fleet verified with osquery.",
  },
  verifying: {
    statusText: "Verifying",
    iconName: "success-partial",
    tooltip: () =>
      "The host acknowledged the MDM command to turn on disk encryption. Fleet is verifying with osquery and retrieving " +
      "the disk encryption key. This may take up to one hour.",
  },
  pending: {
    statusText: "Enforcing (pending)",
    iconName: "pending-partial",
    tooltip: () =>
      "The host will receive the MDM command to turn on disk encryption when the host comes online.",
  },
  failed: {
    statusText: "Failed",
    iconName: "error",
    tooltip: null,
  },
};

interface IOSSettingStatusCellProps {
  status: OsSettingsTableStatusValue;
  operationType: MacMdmProfileOperationType | null;
  profileName: string;
}

const OSSettingStatusCell = ({
  status,
  operationType,
  profileName = "",
}: IOSSettingStatusCellProps) => {
  let displayOption: ProfileDisplayOption = null;

  // windows hosts do not have an operation type at the moment and their display options are
  // different than mac hosts.
  if (!operationType && isMdmProfileStatus(status)) {
    displayOption = WINDOWS_DISK_ENCRYPTION_DISPLAY_CONFIG[status];
  }

  if (operationType) {
    displayOption = PROFILE_DISPLAY_CONFIG[operationType]?.[status];
  }

  const isDeviceUser = window.location.pathname
    .toLowerCase()
    .includes("/device/");

  const isDiskEncryptionProfile =
    profileName === FLEET_FILEVAULT_PROFILE_DISPLAY_NAME;

  if (displayOption) {
    const { statusText, iconName, tooltip } = displayOption;
    const tooltipId = uniqueId();
    return (
      <span className={baseClass}>
        <Icon name={iconName} />
        {tooltip ? (
          <>
            <span
              className="tooltip tooltip__tooltip-icon"
              data-tip
              data-for={tooltipId}
              data-tip-disable={false}
            >
              {statusText}
            </span>
            <ReactTooltip
              place="top"
              effect="solid"
              backgroundColor="#3e4771"
              id={tooltipId}
              data-html
            >
              <span className="tooltip__tooltip-text">
                {status !== "action_required" ? (
                  <TooltipContent
                    innerContent={tooltip}
                    innerProps={{
                      isDiskEncryptionProfile,
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
