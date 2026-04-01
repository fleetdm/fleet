import React from "react";
import { IconNames } from "components/icons";
import Icon from "components/Icon";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

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
        <TooltipWrapper
          tipContent={<span className="tooltip__tooltip-text">{tooltip.tooltipText}</span>}
          position={tooltip.position ?? "bottom"}
          underline={false}
        >
          <span className="tooltip tooltip__tooltip-icon">
            <Button
              onClick={onClick}
              variant="text-link"
              className={`${baseClass}__button`}
            >
              {indicatorText}
            </Button>
          </span>
        </TooltipWrapper>
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
        <TooltipWrapper
          tipContent={<span className="tooltip__tooltip-text">{tooltip.tooltipText}</span>}
          position={tooltip.position ?? "bottom"}
          underline={false}
        >
          <span className="tooltip tooltip__tooltip-icon">
            {indicatorText}
          </span>
        </TooltipWrapper>
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
