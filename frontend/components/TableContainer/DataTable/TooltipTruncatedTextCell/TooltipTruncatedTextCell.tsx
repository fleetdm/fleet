import React, { useRef } from "react";
import classnames from "classnames";

import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import TooltipWrapper from "components/TooltipWrapper";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface ITooltipTruncatedTextCellProps {
  value: React.ReactNode;
  /** Tooltip to display. If this is provided then this will be rendered as the tooltip content. If
   * not, the value will be displayed as the tooltip content. Default: undefined */
  tooltip?: React.ReactNode;
  /** If set to `true` the text inside the tooltip will break on words instead of any character.
   * By default the tooltip text breaks on any character. Default: false */
  tooltipBreakOnWord?: boolean;
  /** @deprecated use the prop `className` in order to add custom classes to this component */
  classes?: string;
  className?: string;
  /** Content does not get truncated */
  prefix?: React.ReactNode;
  /** Content does not get truncated */
  suffix?: React.ReactNode;
}

const baseClass = "tooltip-truncated-cell";

const TooltipTruncatedTextCell = ({
  value,
  tooltip,
  tooltipBreakOnWord = false,
  classes = "w250",
  className,
  prefix,
  suffix,
}: ITooltipTruncatedTextCellProps): JSX.Element => {
  const classNames = classnames(baseClass, classes, className, {
    "tooltip-break-on-word": tooltipBreakOnWord,
  });

  // Tooltip visibility logic: Enable only when text is truncated
  const ref = useRef<HTMLSpanElement>(null);
  const isTruncated = useCheckTruncatedElement(ref);

  const displayValue =
    value === null || value === undefined || value === ""
      ? DEFAULT_EMPTY_CELL_VALUE
      : value;
  const isDefaultValue = displayValue === DEFAULT_EMPTY_CELL_VALUE;

  return (
    <div className={classNames}>
      {prefix && <span className="data-table__prefix">{prefix}</span>}
      <TooltipWrapper
        className="data-table__tooltip-truncated-text-container"
        tipContent={
          <>
            {tooltip ?? displayValue}
            <div className="safari-hack">&nbsp;</div>
            {/* Fixes triple click selecting next element in Safari */}
          </>
        }
        position="top"
        tooltipClass="truncated-tooltip"
        clickable
        showArrow
        delayHide={200}
        disableTooltip={isDefaultValue || !isTruncated}
        underline={false}
      >
        <span
          ref={ref}
          className={`data-table__tooltip-truncated-text ${
            isDefaultValue ? "text-muted" : ""
          } ${isTruncated ? "truncated" : ""}`}
        >
          {displayValue}
        </span>
      </TooltipWrapper>
      {suffix && <span className="data-table__suffix">{suffix}</span>}
    </div>
  );
};

export default TooltipTruncatedTextCell;
