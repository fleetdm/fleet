import React from "react";
import classnames from "classnames";
import ReactTooltip from "react-tooltip";

interface IStatusCellProps {
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

const StatusCell = ({ value, tooltip }: IStatusCellProps): JSX.Element => {
  const classTag = generateClassTag(value);
  const statusClassName = classnames(
    "data-table__status",
    `data-table__status--${classTag}`,
    `status--${classTag}`
  );
  const cellContent = tooltip ? (
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
  return <span className={statusClassName}>{cellContent}</span>;
};

export default StatusCell;
