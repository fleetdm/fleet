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
  tipOffset?: number;
  /** If set to `true`, will not show the tooltip. This can be used to dynamically
   * disable the tooltip from the parent component.
   * @default false
   */
  disableTooltip?: boolean;
  /** If set to `true`, will show the arrow on the tooltip.
   * This can be used to dynamically hide the arrow from the parent component.
   * @default false
   */
  showArrow?: boolean;
  /** Corresponds to the react tooltip 5 `positionStrategy` option - see https://react-tooltip.com/docs/options.
   * Setting as `true` will set the tooltip's `positionStrategy` to `"fixed"`. The default strategy is "absolute".
   * Do this if you run into issues with `overflow: hidden` on the tooltip parent container
   * */
  fixedPositionStrategy?: boolean;
}

const baseClass = "component__tooltip-wrapper";

const TooltipWrapper = ({
  // wrapperCustomClass,
  // elementCustomClass,
  // tipCustomClass,
  children,
  tipContent,
  tipOffset = 5,
  position = "bottom-start",
  isDelayed,
  underline = true,
  className,
  tooltipClass,
  clickable = true,
  disableTooltip = false,
  showArrow = false,
  fixedPositionStrategy = false,
}: ITooltipWrapper) => {
  const wrapperClassNames = classnames(baseClass, className, {
    "show-arrow": showArrow,
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
          noArrow={!showArrow}
          place={position}
          opacity={1}
          disableStyleInjection
          clickable={clickable}
          offset={tipOffset}
          positionStrategy={fixedPositionStrategy ? "fixed" : "absolute"}
        >
          {tipContent}
        </ReactTooltip5>
      )}
    </span>
  );
};

export default TooltipWrapper;
