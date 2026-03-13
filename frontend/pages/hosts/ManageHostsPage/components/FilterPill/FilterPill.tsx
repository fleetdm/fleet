import React, { ReactNode, useRef } from "react";
import classnames from "classnames";

import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import { IconNames } from "components/icons";
import TooltipWrapper from "components/TooltipWrapper";

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
      <span>
        <div className={labelClasses}>
          {icon && <Icon name={icon} />}
          {tooltipContent ? (
            <TooltipWrapper
              showArrow
              tipContent={<span>{tooltipContent}</span>}
              position="top"
              underline={false}
              className={`${baseClass}__tooltip-text`}
            >
              <span ref={pillText}>{label}</span>
            </TooltipWrapper>
          ) : (
            <span className={`${baseClass}__tooltip-text`} ref={pillText}>
              {label}
            </span>
          )}
          <Button
            className={`${baseClass}__clear-filter`}
            onClick={onClear}
            variant="icon"
            title={label}
          >
            <Icon name="close" color="core-fleet-black" size="small" />
          </Button>
        </div>
      </span>
    </div>
  );
};

export default FilterPill;
