import React, { useRef } from "react";
import { uniqueId } from "lodash";
import classnames from "classnames";

import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import TooltipWrapper from "components/TooltipWrapper";

interface ITooltipTruncatedTextCellProps {
  value: React.ReactNode;
  /** Tooltip to display. If this is provided then this will be rendered as the tooltip content. If
   * not, the value will be displayed as the tooltip content. Default: undefined */
  tooltip?: React.ReactNode;
  /** If set to `true` the text inside the tooltip will break on words instead of any character.
   * By default the tooltip text breaks on any character. Default: false */
  tooltipBreakOnWord?: boolean;
  className?: string;
  tooltipPosition?: "top" | "bottom" | "left" | "right";
}

const baseClass = "tooltip-truncated-text";

const TooltipTruncatedText = ({
  value,
  tooltip,
  tooltipBreakOnWord = false,
  className,
  tooltipPosition = "top",
}: ITooltipTruncatedTextCellProps): JSX.Element => {
  const classNames = classnames(baseClass, className, {
    "tooltip-break-on-word": tooltipBreakOnWord,
  });

  // Tooltip visibility logic: Enable only when text is truncated
  const ref = useRef<HTMLInputElement>(null);
  const isTruncated = useCheckTruncatedElement(ref);

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
