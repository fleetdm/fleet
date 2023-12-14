import React from "react";
import classnames from "classnames";
import { uniqueId } from "lodash";

import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";

interface IPerformanceImpactCellProps {
  value: { indicator: string; id: number };
  isHostSpecific?: boolean;
  customIdPrefix?: string;
}

const generateClassTag = (rawValue: string): string => {
  return rawValue.replace(" ", "-").toLowerCase();
};

const PerformanceImpactCell = ({
  value,
  isHostSpecific = false,
  customIdPrefix,
}: IPerformanceImpactCellProps): JSX.Element => {
  const { indicator, id } = value;
  const pillClassName = classnames(
    "data-table__pill",
    `data-table__pill--${generateClassTag(indicator || "")}`,
    "tooltip"
  );

  const disableTooltip = ![
    "Minimal",
    "Considerable",
    "Excessive",
    "Undetermined",
  ].includes(indicator);

  const tooltipText = () => {
    switch (indicator) {
      case "Minimal":
        return (
          <>
            Running this query very <br />
            frequently has little to no <br /> impact on your device&apos;s{" "}
            <br />
            performance.
          </>
        );
      case "Considerable":
        return (
          <>
            Running this query <br /> frequently can have a <br /> noticeable
            impact on your <br /> device&apos;s performance.
          </>
        );
      case "Excessive":
        return (
          <>
            Running this query, even <br /> infrequently, can have a <br />
            significant impact on your <br /> device&apos;s performance.
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
            Performance impact will be available when{" "}
            {isHostSpecific ? "the" : "this"} <br />
            query runs{isHostSpecific && " on this host"}.
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
        data-tip-disable={disableTooltip}
      >
        <span className={pillClassName}>{indicator}</span>
      </span>
      <ReactTooltip
        place="top"
        effect="solid"
        backgroundColor={COLORS["tooltip-bg"]}
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

export default PerformanceImpactCell;
