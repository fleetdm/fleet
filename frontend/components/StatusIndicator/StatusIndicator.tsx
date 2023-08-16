import React from "react";
import classnames from "classnames";
import ReactTooltip from "react-tooltip";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { uniqueId } from "lodash";
import { COLORS } from "styles/var/colors";

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
  let indicatorContent;
  if (tooltip) {
    const tooltipId = uniqueId();
    indicatorContent = (
      <>
        <span
          className="host-status tooltip tooltip__tooltip-icon"
          data-tip
          data-for={`status-${tooltipId}`}
          data-tip-disable={false}
        >
          {value}
        </span>
        <ReactTooltip
          className="status-tooltip"
          place={tooltip?.position ? tooltip.position : "top"}
          type="dark"
          effect="solid"
          id={`status-${tooltipId}`}
          backgroundColor={COLORS["tooltip-bg"]}
        >
          {tooltip.tooltipText}
        </ReactTooltip>
      </>
    );
  } else {
    indicatorContent = <>{value}</>;
  }
  return <span className={indicatorClassNames}>{indicatorContent}</span>;
};

export default StatusIndicator;
