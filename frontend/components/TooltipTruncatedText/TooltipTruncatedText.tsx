import React, { useRef } from "react";
import classnames from "classnames";

import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import TooltipWrapper from "components/TooltipWrapper";

interface ITooltipTruncatedTextCellProps {
  value: React.ReactNode;
  /** Tooltip to display. If this is provided then this will be rendered as the tooltip content. If
   * not, the value will be displayed as the tooltip content. Default: undefined */
  tooltip?: React.ReactNode;
  className?: string;
  tooltipPosition?: "top" | "bottom" | "left" | "right";
  isMobileView?: boolean; // new prop
}

const baseClass = "tooltip-truncated-text";

const TooltipTruncatedText = ({
  value,
  tooltip,
  className,
  tooltipPosition = "top",
  isMobileView = false,
}: ITooltipTruncatedTextCellProps): JSX.Element => {
  const classNames = classnames(baseClass, className);

  // Tooltip visibility logic: Enable only when text is truncated
  const ref = useRef<HTMLInputElement>(null);
  const isTruncated = useCheckTruncatedElement(ref);

  // TODO: RachelPerkins unreleased bug refactor to include mobile tapping/click
  return (
    <TooltipWrapper
      className={classNames}
      disableTooltip={!isTruncated}
      underline={false}
      position={tooltipPosition}
      showArrow
      tipContent={tooltip ?? value}
    >
      <div className={`${baseClass}__text-value`} ref={ref}>
        {value}
      </div>
    </TooltipWrapper>
  );
};

export default TooltipTruncatedText;
