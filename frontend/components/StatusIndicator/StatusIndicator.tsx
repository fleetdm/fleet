import React from "react";
import classnames from "classnames";
import ReactTooltip from "react-tooltip";

interface IStatusIndicatorProps {
  value: string;
  tooltip?: {
    id: number;
    tooltipText: string;
  };
}

const generateClassTag = (rawValue: string): string => {
  if (rawValue === "---") {
    return "indeterminate";
  }
  return rawValue.replace(" ", "-").toLowerCase();
};

const StatusIndicator = ({
  value,
  tooltip,
}: IStatusIndicatorProps): JSX.Element => {
  const classTag = generateClassTag(value);
  const statusClassName = classnames(
    "status-indicator",
    `status-indicator--${classTag}`,
    `status--${classTag}`
  );
  const indicatorContent = tooltip ? (
    <>
      <span
        className="host-status tooltip tooltip__tooltip-icon"
        data-tip
        data-for={`status-${tooltip.id}`}
        data-tip-disable={false}
      >
        {value}
      </span>
      <ReactTooltip
        className="status-tooltip"
        place="top"
        type="dark"
        effect="solid"
        id={`status-${tooltip.id}`}
        backgroundColor="#3e4771"
      >
        {tooltip.tooltipText}
      </ReactTooltip>
    </>
  ) : (
    <>{value}</>
  );
  return <span className={statusClassName}>{indicatorContent}</span>;
};

export default StatusIndicator;
