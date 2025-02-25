import React, { ReactNode } from "react";
import ReactTooltip from "react-tooltip";
import classnames from "classnames";

import Button from "components/buttons/Button";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import { COLORS } from "styles/var/colors";

interface IFilterPillProps {
  label: string;
  onClear: () => void;
  icon?: IconNames;
  tooltipDescription?: string | ReactNode;
  className?: string;
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

  return (
    <div
      className={baseClasses}
      role="status"
      aria-label={`hosts filtered by ${label}`}
    >
      <>
        <span>
          <div className={labelClasses}>
            {icon && <Icon name={icon} />}
            <span
              data-tip={tooltipDescription}
              data-for={`filter-pill-tooltip-${label}`}
            >
              {label}
            </span>
            <Button
              className={`${baseClass}__clear-filter`}
              onClick={onClear}
              variant="small-text-icon"
              title={label}
            >
              <Icon name="close" color="core-fleet-blue" size="small" />
            </Button>
          </div>
        </span>
        {tooltipDescription && (
          <ReactTooltip
            role="tooltip"
            place="bottom"
            effect="solid"
            backgroundColor={COLORS["tooltip-bg"]}
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
