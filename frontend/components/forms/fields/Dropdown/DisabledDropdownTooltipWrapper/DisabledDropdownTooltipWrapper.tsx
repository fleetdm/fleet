import classnames from "classnames";
import React from "react";
import { Tooltip as ReactTooltip5 } from "react-tooltip-5";

import { uniqueId } from "lodash";

interface ITooltipWrapper {
  children: React.ReactNode;
  isDelayed?: boolean;
  // Below two props used here to maintain the API of the old TooltipWrapper
  // A clearer system would be to use the 3 below commented props, which describe exactly where they
  // will apply, `element` being the element this tooltip will wrap. Associated logic is commented
  // out, but ready to be used.
  className?: string;
  tooltipClass?: string;
  // wrapperCustomClass?: string;
  // elementCustomClass?: string;
  // tipCustomClass?: string;
  clickable?: boolean;
  tipContent: React.ReactNode;
}

const baseClass = "disabled-dropdown-tooltip-wrapper";

const TooltipWrapper = ({
  // wrapperCustomClass,
  // elementCustomClass,
  // tipCustomClass,
  children,
  tipContent,
  isDelayed,
  className,
  tooltipClass,
  clickable = true,
}: ITooltipWrapper) => {
  const wrapperClassNames = classnames(baseClass, className, {
    // [`${baseClass}__${wrapperCustomClass}`]: !!wrapperCustomClass,
  });

  const elementClassNames = classnames(`${baseClass}__element`, {
    // [`${baseClass}__${elementCustomClass}`]: !!elementCustomClass,
  });

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
        place="left"
        opacity={1}
        disableStyleInjection
        clickable={clickable}
        offset={24}
        positionStrategy="fixed"
        classNameArrow="tooltip-arrow"
      >
        {tipContent}
      </ReactTooltip5>
    </span>
  );
};

export default TooltipWrapper;
