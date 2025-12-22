import React from "react";
import { PlacesType } from "react-tooltip-5";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "inherited-badge";

interface IInheritedBadgeProps {
  tooltipPosition?: PlacesType;
  tooltipContent: React.ReactNode;
}

const InheritedBadge = ({
  tooltipPosition = "top",
  tooltipContent,
}: IInheritedBadgeProps) => {
  return (
    <div className={baseClass}>
      <TooltipWrapper
        tipContent={tooltipContent}
        showArrow
        position={tooltipPosition}
        tipOffset={8}
        underline={false}
        delayInMs={300} // TODO: Apply pattern of delay tooltip for repeated table tooltips
      >
        <span className={`${baseClass}__element-text`}>Inherited</span>
      </TooltipWrapper>
    </div>
  );
};

export default InheritedBadge;
