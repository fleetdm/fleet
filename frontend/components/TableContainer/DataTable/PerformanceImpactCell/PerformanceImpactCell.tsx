import React from "react";
import classnames from "classnames";
import { uniqueId } from "lodash";

import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";

import { getPerformanceImpactIndicatorTooltip } from "utilities/helpers";
import { PerformanceImpactIndicator } from "interfaces/performance_impact_indicator";

interface IPerformanceImpactCellValue {
  indicator: PerformanceImpactIndicator;
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
          {getPerformanceImpactIndicatorTooltip(indicator, isHostSpecific)}
        </span>
      </ReactTooltip>
    </span>
  );
};

export default PerformanceImpactCell;
