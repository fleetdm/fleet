import classnames from "classnames";
import React, { useLayoutEffect, useRef } from "react";
import { Tooltip as ReactTooltip5, PlacesType } from "react-tooltip-5";

import { uniqueId } from "lodash";

/** Renders tooltip content as-is, but on mount applies `text-wrap: balance`
 * to the tooltip's root element and measures the widest balanced line to set
 * an explicit width on the root — so the tooltip's background hugs the
 * balanced text. CSS alone can't shrink the container: the intrinsic width of
 * a `text-wrap: balance` box is computed as if wrap were `normal`, so it
 * stays at `max-width` even when the balanced text is narrower. */
const BalancedTipContent = ({ children }: { children: React.ReactNode }) => {
  const ref = useRef<HTMLSpanElement>(null);

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return undefined;
    const root = el.parentElement;
    if (!root) return undefined;

    // react-tooltip positions/sizes the tip via floating-ui after mount, so
    // measuring synchronously here can land while the tooltip is still at
    // (0, 0) with an initial width. Defer to the next frame.
    const rafId = requestAnimationFrame(() => {
      // Clear any prior explicit width so wrap uses the mixin's max-width.
      root.style.width = "";
      root.style.textWrap = "balance";
      const range = document.createRange();
      range.selectNodeContents(root);
      // jsdom (Jest) doesn't implement Range.getClientRects, so measurement is
      // a no-op there — balancing is a visual concern with no test coverage
      // to preserve.
      if (typeof range.getClientRects !== "function") return;
      const rects = range.getClientRects();
      // Range.getClientRects returns one rect per text run per line, so a line
      // containing text plus a nested <strong>/<em>/<b> produces multiple
      // narrower rects. Taking the widest single rect would under-measure the
      // line width. Group rects by their top edge (visual line) and compute
      // each line's true width from the leftmost/rightmost extents, then pick
      // the widest line.
      const lineBounds = new Map<number, { left: number; right: number }>();
      for (let i = 0; i < rects.length; i += 1) {
        const rect = rects[i];
        if (rect.width !== 0) {
          // Round to bucket sub-pixel variation on the same visual line.
          const lineKey = Math.round(rect.top);
          const bounds = lineBounds.get(lineKey);
          if (bounds) {
            if (rect.left < bounds.left) bounds.left = rect.left;
            if (rect.right > bounds.right) bounds.right = rect.right;
          } else {
            lineBounds.set(lineKey, { left: rect.left, right: rect.right });
          }
        }
      }
      let widest = 0;
      lineBounds.forEach(({ left, right }) => {
        const lineWidth = right - left;
        if (lineWidth > widest) widest = lineWidth;
      });
      if (widest > 0) {
        const style = window.getComputedStyle(root);
        const padLeft = parseFloat(style.paddingLeft) || 0;
        const padRight = parseFloat(style.paddingRight) || 0;
        root.style.width = `${Math.ceil(widest + padLeft + padRight)}px`;
      }
    });

    return () => cancelAnimationFrame(rafId);
  }, [children]);

  // display: contents so this span leaves no layout box — its children render
  // as direct children of the tooltip root, and `el.parentElement` is that root.
  return (
    <span ref={ref} style={{ display: "contents" }}>
      {children}
    </span>
  );
};

export interface ITooltipWrapper {
  children: React.ReactNode;
  // default is bottom-start
  position?: PlacesType;
  /** A boolean or number defining how long to delay showing the tooltip content on hover over the
   * element. If a boolean, sets delay to the default below. If a number, sets to that
   * many milliseconds. Defaults to `true`, overridden by `delayShowHide` */
  delayShow?: boolean | number;
  /** A boolean or number defining how long to delay hiding the tooltip content on mouseout from the element. If a boolean, sets delay to the default below. If a number, sets to that
   * many milliseconds. Overridden by `delayShowHide`  */
  delayHide?: boolean | number;
  /** A boolean or number defining how long to delay showing and hiding the tooltip content on hover
and mouseout from the element. If a boolean, sets delay to the default below. If a number, sets to that
   * many milliseconds. Overrides `delayShow` and `delayHide` */
  delayShowHide?: boolean | number;
  delayInMs?: number;
  underline?: boolean;
  // Below two props used here to maintain the API of the old TooltipWrapper
  // A clearer system would be to use the 3 below commented props, which describe exactly where they
  // will apply, `element` being the element this tooltip will wrap. Associated logic is commented
  // out, but ready to be used.
  className?: string;
  tooltipClass?: string;
  // wrapperCustomClass?: string;
  // elementCustomClass?: string;
  // tipCustomClass?: string;
  clickable?: boolean;
  tipContent: React.ReactNode;
  tipOffset?: number;
  /** If set to `true`, will not show the tooltip. This can be used to dynamically
   * disable the tooltip from the parent component.
   * @default false
   */
  disableTooltip?: boolean;
  /** If set to `true`, will show the arrow on the tooltip.
   * This can be used to dynamically hide the arrow from the parent component.
   * @default false
   */
  showArrow?: boolean;
  /** Corresponds to the react tooltip 5 `positionStrategy` option - see https://react-tooltip.com/docs/options.
   * Setting as `true` will set the tooltip's `positionStrategy` to `"fixed"`. The default strategy is "absolute".
   * Do this if you run into issues with `overflow: hidden` on the tooltip parent container
   * */
  fixedPositionStrategy?: boolean;
  isMobileView?: boolean;
  /** If `true`, evenly distributes characters across lines and shrinks the
   * tooltip to hug the balanced text so there's no widow word or trailing
   * whitespace on the right. Adds a one-time layout measurement per content
   * change. */
  textBalanced?: boolean;
}

const baseClass = "component__tooltip-wrapper";

const DEFAULT_DELAY_MS = 250;

const TooltipWrapper = ({
  // wrapperCustomClass,
  // elementCustomClass,
  // tipCustomClass,
  children,
  tipContent,
  tipOffset = 5,
  position = "bottom-start",
  delayShow = true,
  delayHide,
  delayShowHide,
  delayInMs, // TODO: Apply pattern of delay tooltip for repeated table tooltips
  underline = true,
  className,
  tooltipClass,
  clickable = true,
  disableTooltip = false,
  showArrow = false,
  fixedPositionStrategy = false,
  isMobileView = false,
  textBalanced = true,
}: ITooltipWrapper) => {
  const wrapperClassNames = classnames(baseClass, className, {
    "show-arrow": showArrow,
    // [`${baseClass}__${wrapperCustomClass}`]: !!wrapperCustomClass,
  });

  const willRenderTooltip = !disableTooltip && !!tipContent;

  const elementClassNames = classnames(`${baseClass}__element`, {
    // [`${baseClass}__${elementCustomClass}`]: !!elementCustomClass,
    [`${baseClass}__underline`]: underline && willRenderTooltip,
  });

  const tipClassNames = classnames(`${baseClass}__tip-text`, tooltipClass, {
    // [`${baseClass}__${tipCustomClass}`]: !!tipCustomClass,
  });

  const tipId = uniqueId();

  let delayShowVal;
  if (typeof delayShow === "boolean" && delayShow) {
    delayShowVal = DEFAULT_DELAY_MS;
  } else if (typeof delayShow === "number") {
    delayShowVal = delayShow;
  }

  let delayHideVal;
  if ((typeof delayHide === "boolean" && delayHide) || clickable) {
    delayHideVal = DEFAULT_DELAY_MS;
  } else if (typeof delayHide === "number") {
    delayHideVal = delayHide;
  }

  if (typeof delayShowHide === "boolean" && delayShowHide) {
    [delayShowVal, delayHideVal] = [DEFAULT_DELAY_MS, DEFAULT_DELAY_MS];
  } else if (typeof delayShowHide === "number") {
    [delayShowVal, delayHideVal] = [delayShowHide, delayShowHide];
  }

  return (
    <span className={wrapperClassNames}>
      <div
        className={elementClassNames}
        data-tip
        data-tooltip-id={tipId}
        style={
          isMobileView && willRenderTooltip ? { cursor: "pointer" } : undefined
        } // With mobile width, show pointer cursor on hover since tooltip won't show on hover
      >
        {children}
      </div>
      {willRenderTooltip && (
        <ReactTooltip5
          className={tipClassNames}
          id={tipId}
          delayShow={delayShowVal || delayInMs}
          delayHide={delayHideVal}
          noArrow={!showArrow}
          place={position}
          opacity={1}
          disableStyleInjection
          clickable={clickable}
          offset={tipOffset}
          positionStrategy={fixedPositionStrategy ? "fixed" : "absolute"}
          globalCloseEvents={
            isMobileView ? { clickOutsideAnchor: true } : undefined
          }
          openEvents={isMobileView ? { click: true } : { mouseenter: true }}
          closeEvents={isMobileView ? { click: true } : { mouseleave: true }}
        >
          {textBalanced ? (
            <BalancedTipContent>{tipContent}</BalancedTipContent>
          ) : (
            tipContent
          )}
        </ReactTooltip5>
      )}
    </span>
  );
};

export default TooltipWrapper;
