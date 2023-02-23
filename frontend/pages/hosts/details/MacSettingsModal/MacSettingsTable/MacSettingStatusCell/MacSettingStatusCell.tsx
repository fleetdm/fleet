import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import {
  MacMdmProfileOperationType,
  MacMdmProfileStatus,
} from "interfaces/mdm";
import _ from "lodash";
import React from "react";
import ReactTooltip from "react-tooltip";

const baseClass = "mac-setting-status-cell";

interface IMacSettingStatusCellProps {
  status: MacMdmProfileStatus;
  operationType: MacMdmProfileOperationType;
}
const MacSettingStatusCell = ({
  status,
  operationType,
}: IMacSettingStatusCellProps): JSX.Element => {
  const PROFILE_DISPLAY_CONFIG = {
    install: {
      pending: {
        statusText: "Enforcing (pending)",
        iconName: "pending",
        tooltipText: "Setting will be enforced when the host comes online.",
      },
      applied: {
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
      applied: null, // should not be reached
      failed: {
        statusText: "Failed",
        iconName: "error",
        tooltipText: undefined,
      },
    },
  } as const;

  const options = PROFILE_DISPLAY_CONFIG[operationType]?.[status];

  if (options) {
    const { statusText, iconName, tooltipText } = options;
    const tooltipId = _.uniqueId();
    return (
      <span className={baseClass}>
        <Icon name={iconName} />
        {tooltipText ? (
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
