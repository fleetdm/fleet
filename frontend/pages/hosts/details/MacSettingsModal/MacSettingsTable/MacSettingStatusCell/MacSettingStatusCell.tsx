import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { IHostMacMdmProfile } from "interfaces/mdm";
import React from "react";
import ReactTooltip from "react-tooltip";

const baseClass = "mac-setting-status-cell";

interface IMacSettingStatusCellProps {
  profile: IHostMacMdmProfile;
}
const MacSettingStatusCell = ({
  profile,
}: IMacSettingStatusCellProps): JSX.Element => {
  const PROFILE_DISPLAY_CONFIG = {
    install: {
      pending: {
        statusText: "Enforcing (pending)",
        iconName: "pending",
        tooltipText: "Setting will be enforced when the host comes online.",
      },
      success: {
        statusText: "Applied",
        iconName: "success",
        tooltipText: "Host applied the setting.",
      },
      failed: {
        statusText: "Failed",
        iconName: "error",
        tooltipText: undefined,
      },
    },
    remove: {
      pending: {
        statusText: "Removing enforcement (pending)",
        iconName: "pending",
        tooltipText: "Enforcement will be removed when the host comes online.",
      },
      success: null, // should not be reached
      failed: {
        statusText: "Failed",
        iconName: "error",
        tooltipText: undefined,
      },
    },
  } as const;

  const { status, operation_type, profile_id: profileId } = profile;
  const options = PROFILE_DISPLAY_CONFIG[operation_type]?.[status];

  if (options) {
    const { statusText, iconName, tooltipText } = options;
    return (
      <span className={baseClass}>
        <Icon name={iconName} />
        {tooltipText ? (
          <>
            <span
              className="tooltip tooltip__tooltip-icon"
              data-tip
              data-for={`${profileId}-status-tooltip`}
              data-tip-disable={false}
            >
              {statusText}
            </span>
            <ReactTooltip
              place="top"
              effect="solid"
              backgroundColor="#3e4771"
              id={`${profileId}-status-tooltip`}
              data-html
            >
              <span className="tooltip__tooltip-text">{tooltipText}</span>
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
