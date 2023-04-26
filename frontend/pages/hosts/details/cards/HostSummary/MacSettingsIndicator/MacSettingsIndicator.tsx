import React from "react";
import ReactTooltip from "react-tooltip";

import { IHostMacMdmProfile, MdmProfileStatus } from "interfaces/mdm";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { IconNames } from "components/icons";

const baseClass = "mac-settings-indicator";

type MacSettingsStatus = "Failing" | "Verifying" | "Pending";

interface IStatusDisplayOption {
  iconName: Extract<
    IconNames,
    "success" | "success-partial" | "pending" | "pending-partial" | "error"
  >;
  tooltipText: string;
}
type StatusDisplayOptions = Record<MacSettingsStatus, IStatusDisplayOption>;

const STATUS_DISPLAY_OPTIONS: StatusDisplayOptions = {
  Verifying: {
    iconName: "success-partial",
    tooltipText: "Host applied the latest settings",
  },
  Pending: {
    iconName: "pending-partial",
    tooltipText: "Host will apply the latest settings when it comes online",
  },
  Failing: {
    iconName: "error",
    tooltipText:
      "Host failed to apply the latest settings. Click to view error(s).",
  },
};

const getMacSettingsStatus = (
  hostMacSettings?: IHostMacMdmProfile[]
): MacSettingsStatus => {
  const statuses = hostMacSettings?.map((setting) => setting.status);
  if (statuses?.includes(MdmProfileStatus.FAILED)) {
    return "Failing";
  }
  if (statuses?.includes(MdmProfileStatus.PENDING)) {
    return "Pending";
  }
  return "Verifying";
};

interface IMacSettingsIndicatorProps {
  profiles: IHostMacMdmProfile[];
  onClick?: () => void;
}
const MacSettingsIndicator = ({
  profiles,
  onClick,
}: IMacSettingsIndicatorProps): JSX.Element => {
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
