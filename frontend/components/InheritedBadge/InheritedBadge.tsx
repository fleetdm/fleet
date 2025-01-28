import { uniqueId } from "lodash";
import React from "react";
import { PlacesType, Tooltip as ReactTooltip5 } from "react-tooltip-5";

const baseClass = "inherited-badge";

interface IInheritedBadgeProps {
  tooltipPosition?: PlacesType;
  tooltipContent: React.ReactNode;
}

const InheritedBadge = ({
  tooltipPosition = "top",
  tooltipContent,
}: IInheritedBadgeProps) => {
  const tooltipId = uniqueId();
  return (
    <div className={baseClass}>
      <span
        className={`${baseClass}__element-text`}
        data-tooltip-id={tooltipId}
      >
        Inherited
      </span>
      <ReactTooltip5
        className={`${baseClass}__tooltip-text`}
        disableStyleInjection
        place={tooltipPosition}
        opacity={1}
        id={tooltipId}
        offset={8}
        positionStrategy="fixed"
      >
        {tooltipContent}
      </ReactTooltip5>
    </div>
  );
};

export default InheritedBadge;
