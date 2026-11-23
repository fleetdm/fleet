import classnames from "classnames";
import { uniqueId } from "lodash";
import React, { useEffect, useState, useRef, useLayoutEffect } from "react";
import { createPortal } from "react-dom";
import ReactTooltip from "react-tooltip";

import { COLORS } from "styles/var/colors";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

interface ITooltipTruncatedTextCellProps {
  value: React.ReactNode;
  /** Tooltip to display. If this is provided then this will be rendered as the tooltip content. If
   * not, the value will be displayed as the tooltip content. Default: undefined */
  tooltip?: React.ReactNode;
  /** If set to `true` the text inside the tooltip will break on words instead of any character.
   * By default the tooltip text breaks on any character. Default: false */
  tooltipBreakOnWord?: boolean;
  className?: string;
  /** Content does not get truncated */
  prefix?: React.ReactNode;
  /** Content does not get truncated */
  suffix?: React.ReactNode;
  /** When `true`, the truncated text grows to fill the available width so
   * `suffix` always sits flush against the right edge of the cell, instead
   * of hugging the end of the (variable-length) text. */
  justifySuffixEnd?: boolean;
  /** When `true`, show the tooltip even when the text is not truncated. Use
   * when the tooltip carries supplemental info (e.g. a raw identifier behind a
   * friendlier display value, or a "+N" suffix implying there's more info)
   * rather than just the truncated text. */
  alwaysShowTooltip?: boolean;
  /** Set when the cell renders inside a link. The tooltip then renders outside the link so clicking it does not
   * navigate, waits before showing so sweeping across rows does not open it, and dismisses on Escape. Default: false */
  isInsideLink?: boolean;
}

const baseClass = "tooltip-truncated-cell";
const LINK_TOOLTIP_SHOW_DELAY_MS = 500;

const TooltipTruncatedTextCell = ({
  value,
  tooltip,
  tooltipBreakOnWord = false,
  className,
  prefix,
  suffix,
  justifySuffixEnd = false,
  alwaysShowTooltip = false,
  isInsideLink = false,
}: ITooltipTruncatedTextCellProps): JSX.Element => {
  const classNames = classnames(baseClass, className, {
    "tooltip-break-on-word": tooltipBreakOnWord,
    "justify-suffix-end": justifySuffixEnd,
  });

  // Tooltip visibility logic: Enable only when text is truncated
  const ref = useRef<HTMLSpanElement>(null);
  const [tooltipDisabled, setTooltipDisabled] = useState(true);

  // Key `prefix`/`suffix` on their presence so JSX callers don't churn the
  // dep identity on every parent render.
  const hasPrefix = Boolean(prefix);
  const hasSuffix = Boolean(suffix);

  useLayoutEffect(() => {
    if (ref?.current !== null) {
      const scrollWidth = ref.current.scrollWidth;
      const offsetWidth = ref.current.offsetWidth;
      setTooltipDisabled(scrollWidth <= offsetWidth);
    }
    // Re-measure whenever a prop that affects the rendered text's width
    // changes. `value` is `React.ReactNode`, so a caller passing JSX creates
    // a new reference on every parent render and this effect will re-run;
    // `setTooltipDisabled` bails when the boolean is unchanged, so this is
    // an extra layout read per row, not a re-render loop.
  }, [value, hasPrefix, hasSuffix, justifySuffixEnd]);
  // End

  // Stable across renders: the show/dismiss state below re-renders while the tooltip is open.
  const [tooltipId] = useState(() => uniqueId());
  const [isTooltipShown, setIsTooltipShown] = useState(false);
  const [isTooltipDismissed, setIsTooltipDismissed] = useState(false);

  useEffect(() => {
    if (!isInsideLink || !isTooltipShown) {
      return undefined;
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setIsTooltipDismissed(true);
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [isInsideLink, isTooltipShown]);

  value =
    value === null || value === undefined || value === ""
      ? DEFAULT_EMPTY_CELL_VALUE
      : value;
  const isDefaultValue = value === DEFAULT_EMPTY_CELL_VALUE;

  const reactTooltip = (
    <ReactTooltip
      place="top"
      effect="solid"
      backgroundColor={COLORS["tooltip-bg"]}
      id={tooltipId}
      data-html
      className={classnames("truncated-tooltip", {
        "truncated-tooltip--dismissed": isTooltipDismissed,
      })} // responsive widths
      clickable
      delayHide={200} // need delay set to hover using clickable
      delayShow={isInsideLink ? LINK_TOOLTIP_SHOW_DELAY_MS : undefined}
      afterShow={isInsideLink ? () => setIsTooltipShown(true) : undefined}
      afterHide={
        isInsideLink
          ? () => {
              setIsTooltipShown(false);
              setIsTooltipDismissed(false);
            }
          : undefined
      }
    >
      <>
        {tooltip ?? value}
        <div className="safari-hack">&nbsp;</div>
        {/* Fixes triple click selecting next element in Safari */}
      </>
    </ReactTooltip>
  );

  return (
    <div className={classNames}>
      {prefix && <span className="data-table__prefix">{prefix}</span>}
      <div
        className="data-table__tooltip-truncated-text-container"
        data-tip
        data-for={tooltipId}
        data-tip-disable={
          isDefaultValue || (tooltipDisabled && !alwaysShowTooltip)
        }
        onMouseEnter={
          isInsideLink ? () => setIsTooltipDismissed(false) : undefined
        }
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
      {isInsideLink
        ? createPortal(
            // React events bubble through portals, so stop clicks here before they reach the link's onClick.
            // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions
            <div
              className={classnames(`${baseClass}-portal`, {
                "tooltip-break-on-word": tooltipBreakOnWord,
              })}
              onClick={(e) => e.stopPropagation()}
            >
              {reactTooltip}
            </div>,
            document.body
          )
        : reactTooltip}
      {suffix && <span className="data-table__suffix">{suffix}</span>}
    </div>
  );
};

export default TooltipTruncatedTextCell;
