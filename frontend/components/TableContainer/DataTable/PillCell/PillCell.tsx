import React from "react";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";

interface IPillCellProps {
  value: [string, number];
}

const CELL_WIDTH = 194;
const CELL_PADDING = 27;

const PILL_WIDTHS: Record<string, number> = {
  Minimal: 75,
  Considerable: 108,
  Excessive: 86,
  Denylisted: 71,
};

const getTooltipOffset = (pillText: string) => {
  const offset: Record<string, number> = {};

  if (PILL_WIDTHS[pillText]) {
    offset.left = CELL_WIDTH / 2 - (PILL_WIDTHS[pillText] / 2 + CELL_PADDING);
  }

  return offset;
};

const generateClassTag = (rawValue: string): string => {
  return rawValue.replace(" ", "-").toLowerCase();
};

const PillCell = (props: IPillCellProps): JSX.Element => {
  const { value } = props;
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
      case "Denylisted":
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
      <div data-tip data-for={id.toString()} data-tip-disable={disable()}>
        <span className={pillClassName}>{pillText}</span>
      </div>
      <ReactTooltip
        place="bottom"
        offset={getTooltipOffset(pillText)}
        type="dark"
        effect="solid"
        backgroundColor="#3e4771"
        id={id.toString()}
        data-html
      >
        <span
          className={`tooltip ${generateClassTag(pillText)}__tooltip-text`}
          style={{ width: "196px" }}
        >
          {tooltipText()}
        </span>
      </ReactTooltip>
    </>
  );
};

export default PillCell;
