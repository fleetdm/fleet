import React from "react";
import classnames from "classnames";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import TooltipWrapper from "components/TooltipWrapper";

interface IStatusIndicatorProps {
  value: string;
  tooltip?: {
    tooltipText: string | JSX.Element;
    position?: "top" | "bottom";
  };
  customIndicatorType?: string;
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
  tooltip,
  customIndicatorType,
}: IStatusIndicatorProps): JSX.Element => {
  const indicatorStateClassTag = generateIndicatorStateClassTag(
    value,
    customIndicatorType
  );
  const indicatorClassNames = classnames(
    "status-indicator",
    `status-indicator--${indicatorStateClassTag}`,
    `status--${indicatorStateClassTag}`
  );
  const indicatorContent = tooltip ? (
    <TooltipWrapper
      position={tooltip?.position ? tooltip.position : "top"}
      showArrow
      underline={false}
      tipContent={tooltip.tooltipText}
      tipOffset={14}
    >
      <span>{value}</span>
    </TooltipWrapper>
  ) : (
    <span>{value}</span>
  );
  return <span className={indicatorClassNames}>{indicatorContent}</span>;
};

export default StatusIndicator;
