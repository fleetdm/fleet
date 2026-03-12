import React from "react";
import classnames from "classnames";

import { getPerformanceImpactIndicatorTooltip } from "utilities/helpers";
import {
  isPerformanceImpactIndicator,
  PerformanceImpactIndicatorValue,
} from "interfaces/schedulable_query";

import TooltipWrapper from "components/TooltipWrapper";

interface IPerformanceImpactCellValue {
  indicator: string;
}
interface IPerformanceImpactCellProps {
  value: IPerformanceImpactCellValue;
  isHostSpecific?: boolean;
}

const generateClassTag = (rawValue: string): string => {
  return rawValue.replace(" ", "-").toLowerCase();
};

const baseClass = "performance-impact-cell";

const PerformanceImpactCell = ({
  value,
  isHostSpecific = false,
}: IPerformanceImpactCellProps): JSX.Element => {
  const { indicator } = value;
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

  return (
    <span className={`${baseClass}`}>
      <TooltipWrapper
        showArrow
        tipContent={
          <span
            className={`tooltip ${generateClassTag(
              indicatorValue || ""
            )}__tooltip-text`}
          >
            {getPerformanceImpactIndicatorTooltip(
              indicatorValue,
              isHostSpecific
            )}
          </span>
        }
        position="top"
        underline={false}
        disableTooltip={disableTooltip}
      >
        <span className={pillClassName}>{indicatorValue}</span>
      </TooltipWrapper>
    </span>
  );
};

export default PerformanceImpactCell;
