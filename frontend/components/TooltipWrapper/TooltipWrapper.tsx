import classnames from "classnames";
import React, { useLayoutEffect, useRef } from "react";
import { Tooltip as ReactTooltip5, PlacesType } from "react-tooltip-5";

import { uniqueId } from "lodash";

/** Renders tooltip content inside a measurable inline-block wrapper: applies
 * `text-wrap: balance`, measures the widest balanced line via Range rects
 * (plus inline replaced elements like SVG/IMG that don't participate in Range
 * text runs), then sets an explicit width on the wrapper so the tooltip's
 * background hugs the balanced text. CSS alone can't shrink the container:
 * the intrinsic width of a `text-wrap: balance` box is computed as if wrap
 * were `normal`, so it stays at `max-width` even when the balanced text is
 * narrower. Applying width to an internal element (rather than the tooltip
 * root) sidesteps react-tooltip-5 rewriting the root's `style` attribute on
 * every position update. The outer tooltip's `width: max-content` then
 * shrinks to this wrapper's explicit width. */
const BalancedTipContent = ({ children }: { children: React.ReactNode }) => {
  const ref = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return undefined;

    // Measurement can race with (a) floating-ui repositioning the tooltip,
    // (b) web-font loading changing bold/italic run widths, and (c) balance
    // needing multiple layout passes to converge as the container shrinks
    // around each measurement. Re-run measurement iteratively until width
    // stabilizes, with a small iteration cap to prevent runaway.
    let disposed = false;
    let lastAppliedWidth = -1;
    let iteration = 0;
    const MAX_ITERATIONS = 6;

    const measure = () => {
      if (disposed) return;
      // Clear our prior explicit width so balance re-runs against the outer
      // tooltip's max-width. Force a reflow via offsetWidth so the browser
      // recomputes the balanced layout before we sample rects.
      el.style.width = "";
      void el.offsetWidth;
      const range = document.createRange();
      range.selectNodeContents(el);
      // jsdom (Jest) doesn't implement Range.getClientRects, so measurement
      // is a no-op there — balancing is a visual concern with no test
      // coverage to preserve.
      if (typeof range.getClientRects !== "function") return;
      // Three widest-line quirks a naive `range.getClientRects()` misses:
      // inline replaced elements (svg/img/…) never appear in Range rects,
      // an inline-flex anchor's `gap` is nowhere in its children's rects,
      // and a flex-centered icon's `top` differs from surrounding text
      // baseline. Fix: also read bounding rects for `svg, img, …, a`, and
      // group by vertical center (not `top`) with a half-line-height fuzz.
      const rects: DOMRect[] = Array.from(range.getClientRects());
      el.querySelectorAll("svg, img, video, canvas, iframe, a").forEach(
        (child) => {
          const rect = child.getBoundingClientRect();
          if (rect.width > 0) rects.push(rect);
        }
      );
      const style = window.getComputedStyle(el);
      const parsedLineHeight = parseFloat(style.lineHeight);
      const fontSize = parseFloat(style.fontSize) || 12;
      const lineHeight = Number.isFinite(parsedLineHeight)
        ? parsedLineHeight
        : fontSize * 1.375;
      const fuzz = lineHeight / 2;
      const centersSorted = rects
        .filter((r) => r.width !== 0)
        .map((r) => ({
          center: r.top + r.height / 2,
          left: r.left,
          right: r.right,
        }))
        .sort((a, b) => a.center - b.center);
      const lines: Array<{
        center: number;
        left: number;
        right: number;
      }> = [];
      for (const item of centersSorted) {
        const last = lines[lines.length - 1];
        if (last && Math.abs(item.center - last.center) < fuzz) {
          if (item.left < last.left) last.left = item.left;
          if (item.right > last.right) last.right = item.right;
        } else {
          lines.push({ ...item });
        }
      }
      let widest = 0;
      for (const line of lines) {
        const lineWidth = line.right - line.left;
        if (lineWidth > widest) widest = lineWidth;
      }
      if (widest <= 0) return;
      const next = Math.ceil(widest);
      // Stop when we've stopped shrinking. Balance can produce slightly
      // narrower widths as the container shrinks around each measurement;
      // we keep going as long as it does, then lock in.
      if (next >= lastAppliedWidth && lastAppliedWidth !== -1) {
        // Restore the previous (narrower) width and stop.
        el.style.width = `${lastAppliedWidth}px`;
        return;
      }
      lastAppliedWidth = next;
      el.style.width = `${next}px`;
      iteration += 1;
      if (iteration < MAX_ITERATIONS) {
        requestAnimationFrame(measure);
      }
    };

    const raf1 = requestAnimationFrame(measure);

    // Fonts loading after mount reflows bold/italic runs. Reset and re-run
    // the convergence loop once fonts.ready resolves. Safe to skip when the
    // API is missing.
    if (
      typeof document !== "undefined" &&
      document.fonts &&
      document.fonts.ready
    ) {
      document.fonts.ready.then(() => {
        if (disposed) return;
        lastAppliedWidth = -1;
        iteration = 0;
        requestAnimationFrame(measure);
      });
    }

    return () => {
      disposed = true;
      cancelAnimationFrame(raf1);
    };
  }, [children]);

  // inline-block so the wrapper has its own measurable box and the tooltip's
  // `width: max-content` shrinks to the wrapper's explicit width once set.
  return (
    <div
      ref={ref}
      style={{ display: "inline-block", textWrap: "balance" }}
    >
      {children}
    </div>
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
