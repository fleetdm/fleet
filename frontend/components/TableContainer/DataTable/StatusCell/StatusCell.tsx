import React from "react";
import classnames from "classnames";
import ReactTooltip from "react-tooltip";

interface IStatusCellProps {
  value: string;
  rowId?: number;
  tooltipText?: string;
}

const generateClassTag = (rawValue: string): string => {
  if (rawValue === "---") {
    return "indeterminate";
  }
  return rawValue.replace(" ", "-").toLowerCase();
};

const StatusCell = ({
  value,
  rowId,
  tooltipText,
}: IStatusCellProps): JSX.Element => {
  const statusClassName = classnames(
    "data-table__status",
    `data-table__status--${generateClassTag(value)}`
  );

  return (
    <span className={statusClassName}>
      {tooltipText ? (
        <>
          <div data-tip data-for={rowId}>
            {value}
          </div>
          <ReactTooltip
            className="online-status-tooltip"
            place="top"
            type="dark"
            effect="solid"
            id={`${rowId}`}
            backgroundColor="#3e4771"
          >
            {tooltipText}
          </ReactTooltip>
        </>
      ) : (
        <>{value}</>
      )}
    </span>
  );
};

export default StatusCell;
