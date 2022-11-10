import React from "react";
import classnames from "classnames";
import ReactTooltip from "react-tooltip";

interface IStatusCellProps {
  value: string;
}

const generateClassTag = (rawValue: string): string => {
  if (rawValue === "---") {
    return "indeterminate";
  }
  return rawValue.replace(" ", "-").toLowerCase();
};

const StatusCell = ({ value }: IStatusCellProps): JSX.Element => {
  console.log(`value: ${value}`);
  const statusClassName = classnames(
    "data-table__status",
    `data-table__status--${generateClassTag(value)}`
  );
  const tipContent = (value: string): string | undefined => {
    switch (value) {
      case "online":
        return "Online hosts will respond to a live query.";
      case "offline":
        return `Offline hosts won't respond to a live query because<br/>
                they may be shut down, asleep, or not connected to<br/>
                the internet.`;
    }
  };

  // TODO: get unique id for each host
  const id = "TESTID";
  return (
    <span className={statusClassName}>
      <div data-tip data-for={id}>
        {value}
      </div>
      <ReactTooltip
        className="online-status-tooltip"
        place="top"
        type="dark"
        effect="solid"
        id={id}
        backgroundColor="#3e4771"
      >
        {tipContent(value)}
        'test-tip-val'
      </ReactTooltip>
    </span>
  );
};

export default StatusCell;
