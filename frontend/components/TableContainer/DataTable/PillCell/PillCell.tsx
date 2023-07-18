import React from "react";
import classnames from "classnames";
import { uniqueId } from "lodash";

import ReactTooltip from "react-tooltip";

interface IPillCellProps {
  value: { indicator: string; id: number };
  customIdPrefix?: string;
  hostDetails?: boolean;
}

const generateClassTag = (rawValue: string): string => {
  return rawValue.replace(" ", "-").toLowerCase();
};

const PillCell = ({
  value,
  customIdPrefix,
  hostDetails,
}: IPillCellProps): JSX.Element => {
  const { indicator, id } = value;
  const pillClassName = classnames(
    "data-table__pill",
    `data-table__pill--${generateClassTag(indicator || "")}`,
    "tooltip"
  );

  const disable = () => {
    switch (indicator) {
      case "Minimal":
        return false;
      case "Considerable":
        return false;
      case "Excessive":
        return false;
      case "Undetermined":
        return false;
      default:
        return true;
    }
  };

  const tooltipText = () => {
    switch (indicator) {
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
      case "Undetermined":
        return (
          <>
            To see performance impact, this query must have run with{" "}
            <b>automations</b> on {hostDetails ? "this" : "at least one"} host.
          </>
        );
      default:
        return null;
    }
  };
  const tooltipId = uniqueId();

  return (
    <>
      <span
        data-tip
        data-for={`${customIdPrefix || "pill"}__${id?.toString() || tooltipId}`}
        data-tip-disable={disable()}
      >
        <span className={pillClassName}>{indicator}</span>
      </span>
      <ReactTooltip
        place="top"
        effect="solid"
        backgroundColor="#3e4771"
        id={`${customIdPrefix || "pill"}__${id?.toString() || tooltipId}`}
        data-html
      >
        <span
          className={`tooltip ${generateClassTag(
            indicator || ""
          )}__tooltip-text`}
        >
          {tooltipText()}
        </span>
      </ReactTooltip>
    </>
  );
};

export default PillCell;
