import React, { ReactNode, useRef, useState } from "react";
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
  const [tooltipContent, setTooltipContent] = useState(tooltipDescription);

  // if tooltipDescription not provided, behave like TooltipTruncatedText
  if (isTruncated && !tooltipContent) {
    setTooltipContent(label);
  }

  const labelWithTooltip = tooltipContent ? (
    <TooltipWrapper
      tipContent={tooltipContent}
      position="top"
      underline={false}
      showArrow
      tipOffset={12}
    >
      <span ref={pillText} className={`${baseClass}__tooltip-text`}>
        {label}
      </span>
    </TooltipWrapper>
  ) : (
    <span ref={pillText} className={`${baseClass}__tooltip-text`}>
      {label}
    </span>
  );

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
            {labelWithTooltip}
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
      </>
    </div>
  );
};

export default FilterPill;
