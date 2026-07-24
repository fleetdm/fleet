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
  isMobileView?: boolean;
  /** Pass-through to TooltipWrapper. Set to `true` when the truncated text
   * lives inside an `overflow: hidden` ancestor — the default `absolute`
   * positioning can misplace the tooltip in that case. */
  fixedPositionStrategy?: boolean;
  /** When `true`, suppress the tooltip even if the text is truncated. Useful
   * when a parent surface owns the hover tooltip. */
  disableTooltip?: boolean;
  /** When `true`, show the tooltip even when the text is not truncated. Use
   * when the tooltip carries supplemental info (e.g. a raw identifier behind a
   * friendlier display value) rather than just the truncated text. */
  alwaysShowTooltip?: boolean;
}

const baseClass = "tooltip-truncated-text";

const TooltipTruncatedText = ({
  value,
  tooltip,
  className,
  tooltipPosition = "top",
  isMobileView = false,
  fixedPositionStrategy = false,
  disableTooltip = false,
  alwaysShowTooltip = false,
}: ITooltipTruncatedTextCellProps): JSX.Element => {
  // Tooltip visibility logic: Enable when text is truncated, or always when
  // `alwaysShowTooltip` is set (supplemental info tooltip).
  const ref = useRef<HTMLInputElement>(null);
  const isTruncated = useCheckTruncatedElement(ref);

  const showTooltip = !disableTooltip && (isTruncated || alwaysShowTooltip);
  // Underline the value to signal a supplemental tooltip is available.
  // Truncation-only tooltips are not underlined.
  const underline = showTooltip && alwaysShowTooltip;

  const classNames = classnames(baseClass, className, {
    [`${baseClass}--underline`]: underline,
  });

  // TODO: RachelPerkins unreleased bug refactor to include mobile tapping/click
  return (
    <TooltipWrapper
      className={classNames}
      disableTooltip={!showTooltip}
      // The underline is applied on the text value via `--underline` (see
      // _styles.scss) instead of TooltipWrapper's own underline, whose
      // negative margin gets clipped by truncating (`overflow: hidden`) parents.
      underline={false}
      position={tooltipPosition}
      showArrow
      tipContent={tooltip ?? value}
      isMobileView={isMobileView}
      fixedPositionStrategy={fixedPositionStrategy}
    >
      <div className={`${baseClass}__text-value`} ref={ref}>
        {value}
      </div>
    </TooltipWrapper>
  );
};

export default TooltipTruncatedText;
