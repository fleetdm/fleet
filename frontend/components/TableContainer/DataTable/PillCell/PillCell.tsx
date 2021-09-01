import React from "react";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";

interface IPillCellProps {
  value: [string, number];
}

const generateClassTag = (rawValue: string): string => {
  return rawValue.replace(" ", "-").toLowerCase();
};

const PillCell = (props: IPillCellProps): JSX.Element => {
  const { value } = props;

  const pillClassName = classnames(
    "data-table__pill",
    `data-table__pill--${generateClassTag(value[0])}`
  );

  const disable = () => {
    switch (value[0]) {
      case "Minimal":
        return false;
      case "Considerate":
        return false;
      case "Excessive":
        return false;
      case "Denylisted":
        return false;
      default:
        return true;
    }
  };

  const tooltipText = () => {
    switch (value[0]) {
      case "Minimal":
        return (
          <>
            Running this query very <br />
            frequently has little to no <br /> impact on your device’s <br />
            performance.
          </>
        );
      case "Considerate":
        return (
          <>
            Running this query <br /> frequently can have a <br /> noticeable
            impact on your <br /> device’s performance.
          </>
        );
      case "Excessive":
        return (
          <>
            Running this query, even <br /> infrequently, can have a <br />
            significant impact on your <br /> device’s performance.
          </>
        );
      case "Denylisted":
        return (
          <>
            This query has been <br /> stopped from running <br /> because of
            excessive <br /> resource consumption.
          </>
        );
      default:
        return null;
    }
  };

  return (
    <>
      <div data-tip data-for={value[1].toString()} data-tip-disable={disable()}>
        <span className={pillClassName}>{value[0]}</span>
      </div>
      <ReactTooltip
        place="bottom"
        type="dark"
        effect="solid"
        backgroundColor="#3e4771"
        id={value[1].toString()}
        data-html
      >
        <span
          className={`tooltip ${generateClassTag(value[0])}__tooltip-text`}
          style={{ width: "196px" }}
        >
          {tooltipText()}
        </span>
      </ReactTooltip>
    </>
  );
};

export default PillCell;
