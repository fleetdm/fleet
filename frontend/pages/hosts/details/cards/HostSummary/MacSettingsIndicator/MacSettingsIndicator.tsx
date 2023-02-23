import React from "react";
import ReactTooltip from "react-tooltip";
import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { IMacSettings, MacSettingsStatus } from "interfaces/mdm";

const baseClass = "mac-settings-indicator";

interface IMacSettingsIndicatorProps {
  profiles: IMacSettings;
  onClick?: () => void;
}
const MacSettingsIndicator = ({
  profiles,
  onClick,
}: IMacSettingsIndicatorProps): JSX.Element => {
  const STATUS_DISPLAY_OPTIONS = {
    Latest: {
      iconName: "success",
      tooltipText: "Host applied the latest settings",
    },
    Pending: {
      iconName: "pending",
      tooltipText: "Host will apply the latest settings when it comes online",
    },
    Failing: {
      iconName: "error",
      tooltipText:
        "Host failed to apply the latest settings. Click to view error(s).",
    },
  } as const;

  const getMacSettingsStatus = (
    hostMacSettings: IMacSettings | undefined
  ): MacSettingsStatus => {
    const statuses = hostMacSettings?.map((setting) => setting.status);
    if (statuses?.includes("failed")) {
      return "Failing";
    }
    if (statuses?.includes("pending")) {
      return "Pending";
    }
    return "Latest";
  };

  const macSettingsStatus = getMacSettingsStatus(profiles);

  const iconName = STATUS_DISPLAY_OPTIONS[macSettingsStatus].iconName;
  const tooltipText = STATUS_DISPLAY_OPTIONS[macSettingsStatus].tooltipText;

  return (
    <span className={`${baseClass} info-flex__data`}>
      <Icon name={iconName} />
      <span
        className="tooltip tooltip__tooltip-icon"
        data-tip
        data-for={`${baseClass}-tooltip`}
        data-tip-disable={false}
      >
        <Button
          onClick={onClick}
          variant="text-link"
          className={`${baseClass}__button`}
        >
          {macSettingsStatus}
        </Button>
      </span>
      <ReactTooltip
        place="bottom"
        effect="solid"
        backgroundColor="#3e4771"
        id={`${baseClass}-tooltip`}
        data-html
      >
        <span className="tooltip__tooltip-text">{tooltipText}</span>
      </ReactTooltip>
    </span>
  );
};

export default MacSettingsIndicator;
