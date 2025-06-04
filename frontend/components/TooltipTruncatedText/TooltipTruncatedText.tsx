import React, { useRef } from "react";
import { uniqueId } from "lodash";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";

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
  const isTruncated = useCheckTruncatedElement(ref);

  const tooltipId = uniqueId();
  return (
    <div className={classNames}>
      <div className="tooltip-truncated" data-tip data-for={tooltipId}>
        <div ref={ref} className={isTruncated ? "truncated" : undefined}>
          {value}
        </div>
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
        disable={!isTruncated}
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
