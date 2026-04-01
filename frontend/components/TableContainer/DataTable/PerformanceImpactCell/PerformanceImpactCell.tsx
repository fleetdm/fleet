import React from "react";
import classnames from "classnames";

import TooltipWrapper from "components/TooltipWrapper";

import {
  isPerformanceImpactIndicator,
  PerformanceImpactIndicatorValue,
} from "interfaces/schedulable_query";

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
  ].includes(indicator);

  const indicatorValue = isPerformanceImpactIndicator(indicator)
    ? indicator
    : PerformanceImpactIndicatorValue.UNDETERMINED;

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
  return (
    <span className={`${baseClass}`}>
      <TooltipWrapper
        tipContent={
          <span className={`tooltip ${generateClassTag(indicator || "")}__tooltip-text`}>
            {tooltipText()}
          </span>
        }
        position="top"
        disableTooltip={disableTooltip}
        underline={false}
      >
        <span className={pillClassName}>{indicatorValue}</span>
      </TooltipWrapper>
    </span>
  );
};

export default PerformanceImpactCell;
