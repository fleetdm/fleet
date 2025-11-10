import classnames from "classnames";
import React from "react";
import { Tooltip as ReactTooltip5, PlacesType } from "react-tooltip-5";

import { uniqueId } from "lodash";

export interface ITooltipWrapper {
  children: React.ReactNode;
  // default is bottom-start
  position?: PlacesType;
  /** A boolean or number defining how long to delay showing the tooltip content on hover over the
   * element. If a boolean, sets delay to the default below. If a number, sets to that
   * many milliseconds. Defaults to `true`, overridden by `delayShowHide` */
  delayShow?: boolean | number;
  /** A boolean or number defining how long to delay hiding the tooltip content on mouseout from the element. If a boolean, sets delay to the default below. If a number, sets to that
   * many milliseconds. Overridden by `delayShowHide`  */
  delayHide?: boolean | number;
  /** A boolean or number defining how long to delay showing and hiding the tooltip content on hover
and mouseout from the element. If a boolean, sets delay to the default below. If a number, sets to that
   * many milliseconds. Overrides `delayShow` and `delayHide` */
  delayShowHide?: boolean | number;
  delayInMs?: number;
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
  isMobileView?: boolean;
}

const baseClass = "component__tooltip-wrapper";

const DEFAULT_DELAY_MS = 250;

const TooltipWrapper = ({
  // wrapperCustomClass,
  // elementCustomClass,
  // tipCustomClass,
  children,
  tipContent,
  tipOffset = 5,
  position = "bottom-start",
  delayShow = true,
  delayHide,
  delayShowHide,
  delayInMs, // TODO: Apply pattern of delay tooltip for repeated table tooltips
  underline = true,
  className,
  tooltipClass,
  clickable = true,
  disableTooltip = false,
  showArrow = false,
  fixedPositionStrategy = false,
  isMobileView = false,
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

  let delayShowVal;
  if (typeof delayShow === "boolean" && delayShow) {
    delayShowVal = DEFAULT_DELAY_MS;
  } else if (typeof delayShow === "number") {
    delayShowVal = delayShow;
  }

  let delayHideVal;
  if (typeof delayHide === "boolean" && delayHide) {
    delayHideVal = DEFAULT_DELAY_MS;
  } else if (typeof delayHide === "number") {
    delayHideVal = delayHide;
  }

  if (typeof delayShowHide === "boolean" && delayShowHide) {
    [delayShowVal, delayHideVal] = [DEFAULT_DELAY_MS, DEFAULT_DELAY_MS];
  } else if (typeof delayShowHide === "number") {
    [delayShowVal, delayHideVal] = [delayShowHide, delayShowHide];
  }

  console.log("isMobileView in TooltipWrapper:", isMobileView);
  return (
    <span className={wrapperClassNames}>
      <div
        className={elementClassNames}
        data-tip
        data-tooltip-id={tipId}
        style={
          isMobileView && !disableTooltip ? { cursor: "pointer" } : undefined
        } // With mobile width, show pointer cursor on hover since tooltip won't show on hover
      >
        {children}
      </div>
      {!disableTooltip && (
        <ReactTooltip5
          className={tipClassNames}
          id={tipId}
          delayShow={delayShowVal || delayInMs}
          delayHide={delayHideVal}
          noArrow={!showArrow}
          place={position}
          opacity={1}
          disableStyleInjection
          clickable={clickable}
          offset={tipOffset}
          positionStrategy={fixedPositionStrategy ? "fixed" : "absolute"}
          globalCloseEvents={
            isMobileView ? { clickOutsideAnchor: true } : undefined
          }
          openEvents={isMobileView ? { click: true } : { mouseenter: true }}
          closeEvents={isMobileView ? { click: true } : { mouseleave: true }}
        >
          {tipContent}
        </ReactTooltip5>
      )}
    </span>
  );
};

export default TooltipWrapper;
