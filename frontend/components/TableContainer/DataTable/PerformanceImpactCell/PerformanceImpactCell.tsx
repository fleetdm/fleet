import React from "react";
import classnames from "classnames";
import { uniqueId } from "lodash";

import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";

interface IPerformanceImpactCellValue {
  indicator: string;
  id?: number;
}
interface IPerformanceImpactCellProps {
  value: IPerformanceImpactCellValue;
  isHostSpecific?: boolean;
  customIdPrefix?: string;
}

const generateClassTag = (rawValue: string): string => {
  return rawValue.replace(" ", "-").toLowerCase();
};

const baseClass = "performance-impact-cell";

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
    "Denylisted",
  ].includes(indicator);

  const tooltipText = () => {
    switch (indicator) {
      case "Minimal":
        return (
          <>
            Running this query very frequently has little to no <br /> impact on
            your device&apos;s performance.
          </>
        );
      case "Considerable":
        return (
          <>
            Running this query frequently can have a noticeable <br />
            impact on your device&apos;s performance.
          </>
        );
      case "Excessive":
        return (
          <>
            Running this query, even infrequently, can have a <br />
            significant impact on your device&apos;s performance.
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
    <span className={`${baseClass}`}>
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
    </span>
  );
};

export default PerformanceImpactCell;
