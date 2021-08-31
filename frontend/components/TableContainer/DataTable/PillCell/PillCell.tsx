import React from "react";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";

interface IPillCellProps {
  value: string;
}

const generateClassTag = (rawValue: string): string => {
  return rawValue.replace(" ", "-").toLowerCase();
};

const PillCell = (props: IPillCellProps): JSX.Element => {
  const { value } = props;

  console.log(value);

  const pillClassName = classnames(
    "data-table__pill",
    `data-table__pill--${generateClassTag(value)}`
  );

  const disable = () => {
    switch (value) {
      case "undetermined":
        return false;
      case "minimal":
        return false;
      case "considerable":
        return false;
      case "excessive":
        return false;
      case "denylisted":
        return false;
      default:
        return true;
    }
  };

  const tooltipText = () => {
    console.log("value", value);
    switch (value) {
      case "undetermined":
        return "Running this query very frequently has little to no impact on your device’s performance.";
      case "considerable":
        return "Running this query frequently can have a noticeable impact on your device’s performance.";
      case "excessive":
        return "Running this query, even infrequently, can have a significant impact on your device’s performance.";
      case "denylisted":
        return "This query has been stopped from running because of excessive resource consumption.";
      default:
        return null;
    }
  };

  console.log(pillClassName);

  return (
    <>
      <div data-tip data-for="value" data-tip-disable={disable()}>
        <span className={pillClassName}>{value}</span>
      </div>
      <ReactTooltip
        place="bottom"
        type="dark"
        effect="solid"
        backgroundColor="#3e4771"
        id="value"
        data-html
      >
        <span
          className={`tooltip ${generateClassTag(value)}__tooltip-text`}
          style={{ width: `196px` }}
        >
          {tooltipText()}
        </span>
      </ReactTooltip>
    </>
  );
};

export default PillCell;
