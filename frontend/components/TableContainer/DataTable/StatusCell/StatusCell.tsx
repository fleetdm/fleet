import React from "react";
import classnames from "classnames";
import ReactTooltip from "react-tooltip";

interface IStatusCellProps {
  value: string;
  tooltip?: {
    rowId?: number;
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
  const statusClassName = classnames(
    "data-table__status",
    `data-table__status--${generateClassTag(value)}`
  );
  const cellContent = tooltip ? (
    <>
      <div data-tip data-for={tooltip.rowId}>
        {value}
      </div>
      <ReactTooltip
        className="status-tooltip"
        place="top"
        type="dark"
        effect="solid"
        id={`${tooltip.rowId}`}
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
