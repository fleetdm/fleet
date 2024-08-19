import classnames from "classnames";
import React from "react";
import { Tooltip as ReactTooltip5 } from "react-tooltip-5";

import { uniqueId } from "lodash";

interface IDropdownOptionTooltipWrapper {
  children: React.ReactNode;
  isDelayed?: boolean;
  className?: string;
  tooltipClass?: string;
  clickable?: boolean;
  tipContent: React.ReactNode;
  /** Location defaults to left */
  place?: "left" | "right" | "top" | "bottom";
  offset?: number;
}

const baseClass = "dropdown-option-tooltip-wrapper";

const DropdownOptionTooltipWrapper = ({
  children,
  tipContent,
  isDelayed,
  className,
  tooltipClass,
  clickable = true,
  place = "left",
  offset = 24,
}: IDropdownOptionTooltipWrapper) => {
  const wrapperClassNames = classnames(baseClass, className);

  const elementClassNames = classnames(`${baseClass}__element`);

  const tipClassNames = classnames(
    `${baseClass}__tip-text`,
    `${baseClass}__dropdown-tooltip-arrow`,
    tooltipClass
  );

  const tipId = uniqueId();

  return (
    <span className={wrapperClassNames}>
      <div className={elementClassNames} data-tooltip-id={tipId}>
        {children}
      </div>
      <ReactTooltip5
        className={tipClassNames}
        id={tipId}
        delayShow={isDelayed ? 500 : undefined}
        delayHide={isDelayed ? 500 : undefined}
        place={place}
        opacity={1}
        disableStyleInjection
        clickable={clickable}
        offset={offset}
        positionStrategy="fixed"
      >
        {tipContent}
      </ReactTooltip5>
    </span>
  );
};

export default DropdownOptionTooltipWrapper;
