import React from "react";
import classnames from "classnames";
import ReactTooltip from "react-tooltip";

interface IStatusCellProps {
  value: string;
  tooltipInfo?: {
    rowId: number;
    tooltipText: string;
  };
}

const generateClassTag = (rawValue: string): string => {
  if (rawValue === "---") {
    return "indeterminate";
  }
  return rawValue.replace(" ", "-").toLowerCase();
};

const StatusCell = ({ value, tooltipInfo }: IStatusCellProps): JSX.Element => {
  const statusClassName = classnames(
    "data-table__status",
    `data-table__status--${generateClassTag(value)}`
  );
  const cellContent = tooltipInfo ? (
    <>
      <div data-tip={tooltipInfo.tooltipText} data-for={tooltipInfo.rowId}>
        {value}
      </div>
      <ReactTooltip
        className="online-status-tooltip"
        place="top"
        type="dark"
        effect="solid"
        id={`${tooltipInfo.rowId}`}
        backgroundColor="#3e4771"
      />
      {/* {tooltipInfo.tooltipText}
        </ReactTooltip> */}
    </>
  ) : (
    <>{value}</>
  );
  return <span className={statusClassName}>{cellContent}</span>;
};

export default StatusCell;
