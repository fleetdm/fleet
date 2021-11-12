import React from "react";
import classnames from "classnames";
import { uniqueId } from "lodash";

import ReactTooltip from "react-tooltip";

interface IPillCellProps {
  value: [string, number];
  customIdPrefix?: string;
}

const generateClassTag = (rawValue: string): string => {
  return rawValue.replace(" ", "-").toLowerCase();
};

const PillCell = ({ value, customIdPrefix }: IPillCellProps): JSX.Element => {
  const [pillText, id] = value;

  const pillClassName = classnames(
    "data-table__pill",
    `data-table__pill--${generateClassTag(pillText)}`
  );

  const disable = () => {
    switch (pillText) {
      case "Minimal":
        return false;
      case "Considerable":
        return false;
      case "Excessive":
        return false;
      default:
        return true;
    }
  };

  const tooltipText = () => {
    switch (pillText) {
      case "Minimal":
        return (
          <>
            Running this query very <br />
            frequently has little to no <br /> impact on your device’s <br />
            performance.
          </>
        );
      case "Considerable":
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
      <span
        data-tip
        data-for={`${customIdPrefix || "pill"}__${
          id?.toString() || uniqueId()
        }`}
        data-tip-disable={disable()}
      >
        <span className={pillClassName}>{pillText}</span>
      </span>
      <ReactTooltip
        place="bottom"
        // offset={getTooltipOffset(pillText)}
        type="dark"
        effect="solid"
        backgroundColor="#3e4771"
        id={`${customIdPrefix || "pill"}__${id?.toString() || uniqueId()}`}
        data-html
      >
        <span
          className={`tooltip ${generateClassTag(pillText)}__tooltip-text`}
          style={{ textAlign: "center" }}
        >
          {tooltipText()}
        </span>
      </ReactTooltip>
    </>
  );
};

export default PillCell;
