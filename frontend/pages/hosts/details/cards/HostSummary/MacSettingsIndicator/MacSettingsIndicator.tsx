import React from "react";
import ReactTooltip from "react-tooltip";

import { IHostMacMdmProfile } from "interfaces/mdm";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { IconNames } from "components/icons";

const baseClass = "mac-settings-indicator";

type MacProfileStatus = "Failed" | "Verifying" | "Pending" | "Verified";

interface IStatusDisplayOption {
  iconName: Extract<
    IconNames,
    "success" | "success-partial" | "pending" | "pending-partial" | "error"
  >;
  tooltipText: string;
}
type StatusDisplayOptions = Record<MacProfileStatus, IStatusDisplayOption>;

const STATUS_DISPLAY_OPTIONS: StatusDisplayOptions = {
  Verified: {
    iconName: "success",
    tooltipText:
      "The host applied all OS settings. Fleet verified with osquery.",
  },
  Verifying: {
    iconName: "success-partial",
    tooltipText:
      "The host acknowledged all MDM commands to apply OS settings. " +
      "Fleet is verifying the OS settings are applied with osquery.",
  },
  Pending: {
    iconName: "pending-partial",
    tooltipText:
      "The host will receive MDM command to apply OS settings when the host comes online.",
  },
  Failed: {
    iconName: "error",
    tooltipText:
      "The host failed to apply the latest OS settings. Click to view error(s).",
  },
};

/**
 * Returns the displayed status of the macOS settings field based on the
 * profile statuses.
 * If any profile has a status of "failed", the status will be displayed as "Failed" and
 * continues to fall through to "Pending" and "Verifying" if any profiles have those statuses.
 * Finally if all profiles have a status of "verified", the status will be displayed as "Verified".
 */
const getMacProfileStatus = (
  hostMacSettings: IHostMacMdmProfile[]
): MacProfileStatus => {
  const statuses = hostMacSettings.map((setting) => setting.status);
  if (statuses.includes("failed")) {
    return "Failed";
  }
  if (statuses.includes("pending")) {
    return "Pending";
  }
  if (statuses.includes("verifying")) {
    return "Verifying";
  }
  return "Verified";
};

interface IMacSettingsIndicatorProps {
  profiles: IHostMacMdmProfile[];
  onClick?: () => void;
}
const MacSettingsIndicator = ({
  profiles,
  onClick,
}: IMacSettingsIndicatorProps): JSX.Element => {
  const macProfileStatus = getMacProfileStatus(profiles);

  const statusDisplayOption = STATUS_DISPLAY_OPTIONS[macProfileStatus];

  return (
    <span className={`${baseClass} info-flex__data`}>
      <Icon name={statusDisplayOption.iconName} />
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
          {macProfileStatus}
        </Button>
      </span>
      <ReactTooltip
        place="bottom"
        effect="solid"
        backgroundColor="#3e4771"
        id={`${baseClass}-tooltip`}
        data-html
      >
        <span className="tooltip__tooltip-text">
          {statusDisplayOption.tooltipText}
        </span>
      </ReactTooltip>
    </span>
  );
};

export default MacSettingsIndicator;
