import React from "react";
import classnames from "classnames";
import ReactTooltip from "react-tooltip";

interface IStatusCellProps {
  rowId: number;
  value: string;
}

const generateClassTag = (rawValue: string): string => {
  if (rawValue === "---") {
    return "indeterminate";
  }
  return rawValue.replace(" ", "-").toLowerCase();
};

const StatusCell = ({ rowId, value }: IStatusCellProps): JSX.Element => {
  const statusClassName = classnames(
    "data-table__status",
    `data-table__status--${generateClassTag(value)}`
  );
  const tipContent = (tipText: string): string | undefined => {
    switch (tipText) {
      case "online":
        return "Online hosts will respond to a live query.";
      case "offline":
        return `Offline hosts won't respond to a live query because
                they may be shut down, asleep, or not connected to
                the internet.`;
      default:
        return "";
    }
  };

  return (
    <span className={statusClassName}>
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
        {tipContent(value)}
      </ReactTooltip>
    </span>
  );
};

export default StatusCell;
