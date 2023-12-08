import React from "react";
import ReactTooltip from "react-tooltip";
import { IconNames } from "components/icons";
import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { COLORS } from "styles/var/colors";

const baseClass = "profile-status-indicator";

export interface IProfileStatusIndicatorProps {
  indicatorText: string;
  iconName: IconNames;
  onClick?: () => void;
  tooltip?: {
    tooltipText: string | null;
    position?: "top" | "bottom";
  };
}

const ProfileStatusIndicator = ({
  indicatorText,
  iconName,
  onClick,
  tooltip,
}: IProfileStatusIndicatorProps) => {
  const getIndicatorTextWrapped = () => {
    if (onClick && tooltip?.tooltipText) {
      return (
        <>
          <span
            className="tooltip tooltip__tooltip-icon"
            data-tip
            data-for={`${indicatorText}-tooltip`}
            data-tip-disable={false}
          >
            <Button
              onClick={onClick}
              variant="text-link"
              className={`${baseClass}__button`}
            >
              {indicatorText}
            </Button>
          </span>
          <ReactTooltip
            place={tooltip.position ?? "bottom"}
            effect="solid"
            backgroundColor={COLORS["tooltip-bg"]}
            id={`${indicatorText}-tooltip`}
            data-html
          >
            <span className="tooltip__tooltip-text">{tooltip.tooltipText}</span>
          </ReactTooltip>
        </>
      );
    }

    // onclick without tooltip
    if (onClick) {
      return (
        <Button
          onClick={onClick}
          variant="text-link"
          className={`${baseClass}__button`}
        >
          {indicatorText}
        </Button>
      );
    }

    // tooltip without onclick
    if (tooltip?.tooltipText) {
      return (
        <>
          <span
            className="tooltip tooltip__tooltip-icon"
            data-tip
            data-for={`${indicatorText}-tooltip`}
            data-tip-disable={false}
          >
            {indicatorText}
          </span>
          <ReactTooltip
            place={tooltip.position ?? "bottom"}
            effect="solid"
            backgroundColor={COLORS["tooltip-bg"]}
            id={`${indicatorText}-tooltip`}
            data-html
          >
            <span className="tooltip__tooltip-text">{tooltip.tooltipText}</span>
          </ReactTooltip>
        </>
      );
    }

    // no tooltip, no onclick
    return indicatorText;
  };

  return (
    <span className={`${baseClass} info-flex__data`}>
      <Icon name={iconName} />
      {getIndicatorTextWrapped()}
    </span>
  );
};

export default ProfileStatusIndicator;
