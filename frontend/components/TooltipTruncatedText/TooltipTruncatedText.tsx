import React, { useState, useRef, useLayoutEffect } from "react";
import { uniqueId } from "lodash";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";

interface ITooltipTruncatedTextCellProps {
  value: React.ReactNode;
  /** Tooltip to display. If this is provided then this will be rendered as the tooltip content. If
   * not, the value will be displayed as the tooltip content. Default: undefined */
  tooltip?: React.ReactNode;
  /** If set to `true` the text inside the tooltip will break on words instead of any character.
   * By default the tooltip text breaks on any character. Default: false */
  tooltipBreakOnWord?: boolean;
  className?: string;
}

const baseClass = "tooltip-truncated-text";

const TooltipTruncatedText = ({
  value,
  tooltip,
  tooltipBreakOnWord = false,
  className,
}: ITooltipTruncatedTextCellProps): JSX.Element => {
  const classNames = classnames(baseClass, className, {
    "tooltip-break-on-word": tooltipBreakOnWord,
  });

  // Tooltip visibility logic: Enable only when text is truncated
  const ref = useRef<HTMLInputElement>(null);
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
  return (
    <div ref={ref} className={classNames}>
      <div className={"tooltip-truncated"} data-tip data-for={tooltipId}>
        <span className={tooltipDisabled ? "" : "truncated"}>{value}</span>
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
        disable={tooltipDisabled}
      >
        <>
          {tooltip ?? value}
          <div className="safari-hack">&nbsp;</div>
          {/* Fixes triple click selecting next element in Safari */}
        </>
      </ReactTooltip>
    </div>
  );
};

export default TooltipTruncatedText;
