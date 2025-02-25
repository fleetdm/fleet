import React, { ReactNode } from "react";
import classnames from "classnames";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import TooltipWrapper from "components/TooltipWrapper";
import { capitalize } from "lodash";

const baseClass = "status-indicator";

export type IIndicatorValue = "success" | "warning" | "error" | "indeterminate";

interface IStatusIndicatorProps {
  /** Only the first letter of value will be capitalized by the component.
   * NOTE: Do not rely on the value prop to determine the status indicator. Use the
   * `indicator` prop instead.
   */
  value: string;
  /** The indicator type allows for showing the desired indicator.
   * NOTE: use this instead relying on the `value` prop to determine the indicator.
   */
  indicator?: IIndicatorValue;
  tooltip?: {
    tooltipText: ReactNode;
    position?: "top" | "bottom";
  };
  /**
   * @deprecated Use `indicator` instead to show the desired indicator.
   */
  customIndicatorType?: string;
  className?: string;
}

const generateIndicatorStateClassTag = (
  rawValue: string,
  customIndicatorType?: string
): string => {
  if (rawValue === DEFAULT_EMPTY_CELL_VALUE) {
    return "indeterminate";
  }
  const prefix = customIndicatorType ? `${customIndicatorType}-` : "";
  return `${prefix}${rawValue.replace(" ", "-").toLowerCase()}`;
};

const StatusIndicator = ({
  value,
  indicator,
  tooltip,
  customIndicatorType,
  className,
}: IStatusIndicatorProps): JSX.Element => {
  const indicatorStateClassTag = generateIndicatorStateClassTag(
    value,
    customIndicatorType
  );

  const classes = classnames(
    baseClass,
    className,
    `${baseClass}--${indicatorStateClassTag}`,
    `status--${indicatorStateClassTag}`,
    indicator ? `${baseClass}--${indicator}` : null
  );

  const capitalizedValue = capitalize(value);

  const indicatorContent = tooltip ? (
    <TooltipWrapper
      position={tooltip?.position ? tooltip.position : "top"}
      showArrow
      underline={false}
      tipContent={tooltip.tooltipText}
      tipOffset={14}
    >
      {capitalizedValue}
    </TooltipWrapper>
  ) : (
    capitalizedValue
  );

  return <span className={classes}>{indicatorContent}</span>;
};

export default StatusIndicator;
