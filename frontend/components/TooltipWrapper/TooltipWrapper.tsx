import classnames from "classnames";
import React from "react";
import { Tooltip as ReactTooltip5, PlacesType } from "react-tooltip-5";

import { uniqueId } from "lodash";

interface ITooltipWrapper {
  children: React.ReactNode;
  // default is bottom-start
  position?: PlacesType;
  isDelayed?: boolean;
  underline?: boolean;
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
  /** If set to `true`, will not show the tooltip. This can be used to dynamically
   * disable the tooltip from the parent component.
   *
   * @default false
   */
  disableTooltip?: boolean;
}

const baseClass = "component__tooltip-wrapper";

const TooltipWrapper = ({
  // wrapperCustomClass,
  // elementCustomClass,
  // tipCustomClass,
  children,
  tipContent,
  position = "bottom-start",
  isDelayed,
  underline = true,
  className,
  tooltipClass,
  clickable = true,
  disableTooltip = false,
}: ITooltipWrapper) => {
  const wrapperClassNames = classnames(baseClass, className, {
    // [`${baseClass}__${wrapperCustomClass}`]: !!wrapperCustomClass,
  });

  const elementClassNames = classnames(`${baseClass}__element`, {
    // [`${baseClass}__${elementCustomClass}`]: !!elementCustomClass,
    [`${baseClass}__underline`]: underline,
  });

  const tipClassNames = classnames(`${baseClass}__tip-text`, tooltipClass, {
    // [`${baseClass}__${tipCustomClass}`]: !!tipCustomClass,
  });

  const tipId = uniqueId();

  return (
    <span className={wrapperClassNames}>
      <div className={elementClassNames} data-tooltip-id={tipId}>
        {children}
      </div>
      {!disableTooltip && (
        <ReactTooltip5
          className={tipClassNames}
          id={tipId}
          delayShow={isDelayed ? 500 : undefined}
          delayHide={isDelayed ? 500 : undefined}
          noArrow
          place={position}
          opacity={1}
          disableStyleInjection
          clickable={clickable}
          offset={5}
        >
          {tipContent}
        </ReactTooltip5>
      )}
    </span>
  );
};

export default TooltipWrapper;
