import React, { ReactNode, useRef } from "react";
import ReactTooltip from "react-tooltip";
import classnames from "classnames";

import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import { COLORS } from "styles/var/colors";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import { IconNames } from "components/icons";

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

  const pillText = useRef(null);
  const isTruncated = useCheckTruncatedElement(pillText);

  // if tooltipDescription not provided, behave like TooltipTruncatedText
  const tooltipContent =
    tooltipDescription ?? (isTruncated ? label : undefined);

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
              data-tip={tooltipContent}
              data-for={`filter-pill-tooltip-${label}`}
              className={`${baseClass}__tooltip-text`}
              ref={pillText}
            >
              {label}
            </span>
            <Button
              className={`${baseClass}__clear-filter`}
              onClick={onClear}
              variant="icon"
              title={label}
            >
              <Icon name="close" color="core-fleet-blue" size="small" />
            </Button>
          </div>
        </span>
        {tooltipContent && (
          <ReactTooltip
            role="tooltip"
            place="top"
            effect="solid"
            backgroundColor={COLORS["tooltip-bg"]}
            id={`filter-pill-tooltip-${label}`}
            data-html
          >
            <span>{tooltipContent}</span>
          </ReactTooltip>
        )}
      </>
    </div>
  );
};

export default FilterPill;
