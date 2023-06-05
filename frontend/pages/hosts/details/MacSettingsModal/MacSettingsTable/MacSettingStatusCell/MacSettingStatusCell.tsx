import React from "react";
import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { MacMdmProfileOperationType } from "interfaces/mdm";

import { MacSettingsTableStatusValue } from "../MacSettingsTableConfig";
import TooltipContent, {
  TooltipInnerContentFunc,
  TooltipInnerContentOption,
} from "./components/Tooltip/TooltipContent";
import TooltipInnerContentActionRequired from "./components/Tooltip/ActionRequired";

const baseClass = "mac-setting-status-cell";

type ProfileDisplayOption = {
  statusText: string;
  iconName: IconNames;
  tooltip: TooltipInnerContentOption | null;
} | null;

type OperationTypeOption = Record<
  MacSettingsTableStatusValue,
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
      tooltip: "Setting will be enforced when the host comes online.", // TODO: this doesn't work for disk encryption or the device page generally
    },
    action_required: {
      statusText: "Action required (pending)",
      iconName: "pending-partial",
      tooltip: TooltipInnerContentActionRequired as TooltipInnerContentFunc,
    },
    verifying: {
      statusText: "Verifying",
      iconName: "success-partial",
      tooltip: "Host applied the setting.",
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
      tooltip: "Enforcement will be removed when the host comes online.",
    },
    action_required: null, // should not be reached
    verifying: null, // should not be reached
    failed: {
      statusText: "Failed",
      iconName: "error",
      tooltip: null,
    },
  },
};

interface IMacSettingStatusCellProps {
  status: MacSettingsTableStatusValue;
  operationType: MacMdmProfileOperationType;
  profileName?: string;
}

const MacSettingStatusCell = ({
  status,
  operationType,
  profileName = "",
}: IMacSettingStatusCellProps): JSX.Element => {
  const options = PROFILE_DISPLAY_CONFIG[operationType]?.[status];
  // TODO: confirm this approach
  const isDeviceUser = window.location.pathname
    .toLowerCase()
    .includes("/device/");

  if (options) {
    const { statusText, iconName, tooltip } = options;
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
                <TooltipContent
                  innerContent={tooltip}
                  innerProps={{ isDeviceUser, profileName }}
                />
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
export default MacSettingStatusCell;
