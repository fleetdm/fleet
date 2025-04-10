import React, { useState, useRef, useLayoutEffect } from "react";
import { uniqueId } from "lodash";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { COLORS } from "styles/var/colors";

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
  const [tooltipDisabled, setTooltipDisabled] = useState(true);

  useLayoutEffect(() => {
    if (ref?.current !== null) {
      const scrollWidth = ref.current.scrollWidth;
      const offsetWidth = ref.current.offsetWidth;
      setTooltipDisabled(scrollWidth <= offsetWidth);
    }
  }, [ref]);
  // End

  const tooltipId = uniqueId();
  const isDefaultValue = value === DEFAULT_EMPTY_CELL_VALUE;

  return (
    <div className={classNames}>
      {prefix && <span className="data-table__prefix">{prefix}</span>}
      <div
        className="data-table__tooltip-truncated-text-container"
        data-tip
        data-for={tooltipId}
        data-tip-disable={isDefaultValue || tooltipDisabled}
      >
        <span
          ref={ref}
          className={`data-table__tooltip-truncated-text ${
            isDefaultValue ? "text-muted" : ""
          } ${tooltipDisabled ? "" : "truncated"}`}
        >
          {value}
        </span>
      </div>
      <ReactTooltip
        place="top"
        effect="solid"
        backgroundColor={COLORS["tooltip-bg"]}
        id={tooltipId}
        data-html
        className="truncated-tooltip" // responsive widths
        clickable
        delayHide={200} // need delay set to hover using clickable
      >
        <>
          {tooltip ?? value}
          <div className="safari-hack">&nbsp;</div>
          {/* Fixes triple click selecting next element in Safari */}
        </>
      </ReactTooltip>
      {suffix && <span className="data-table__suffix">{suffix}</span>}
    </div>
  );
};

export default TooltipTruncatedTextCell;
