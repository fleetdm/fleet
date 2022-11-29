import React, { ReactNode } from "react";
import ReactTooltip from "react-tooltip";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

interface IFilterPillProps {
  label: string;
  icon?: string;
  tooltipDescription?: string | ReactNode;
  className?: string;
  onClear: () => void;
}

const baseClass = "filter-pill";

const FilterPill = ({
  label,
  icon,
  tooltipDescription,
  className,
  onClear,
}: IFilterPillProps) => {
  const baseClasses = classnames(baseClass, className);
  const labelClasses = classnames(`${baseClass}__label`, {
    tooltip: tooltipDescription !== undefined && tooltipDescription !== "",
  });
  console.log("icon", icon);
  return (
    <div
      className={baseClasses}
      role="status"
      aria-label={`hosts filtered by ${label}`}
    >
      <>
        <span
          data-tip={tooltipDescription}
          data-for={`filter-pill-tooltip-${label}`}
        >
          <div className={labelClasses}>
            {icon && <Icon name="policy" />}
            {label}
            <Button
              className={`${baseClass}__clear-filter`}
              onClick={onClear}
              variant={"small-text-icon"}
              title={label}
            >
              <Icon name="ex" color="core-fleet-blue" />
            </Button>
          </div>
        </span>
        {tooltipDescription && (
          <ReactTooltip
            role="tooltip"
            place="bottom"
            effect="solid"
            backgroundColor="#3e4771"
            id={`filter-pill-tooltip-${label}`}
            data-html
          >
            <span>{tooltipDescription}</span>
          </ReactTooltip>
        )}
      </>
    </div>
  );
};

export default FilterPill;
