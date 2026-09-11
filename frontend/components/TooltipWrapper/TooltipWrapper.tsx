import classnames from "classnames";
import React, { useLayoutEffect, useRef } from "react";
import { Tooltip as ReactTooltip5, PlacesType } from "react-tooltip-5";

import { uniqueId } from "lodash";

/** Shrinks the tooltip to hug balanced text. `text-wrap: balance` alone
 * leaves the container at `max-width` (intrinsic width is computed as
 * wrap: normal), and setting width on the tooltip root gets wiped by
 * react-tooltip-5's per-render style spread — so measure widest line on
 * an inline-block child and set width there. */
const BalancedTipContent = ({ children }: { children: React.ReactNode }) => {
  const ref = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return undefined;

    // Convergence loop: balance re-runs as the container shrinks around
    // each measurement, so iterate until width stops shrinking.
    let disposed = false;
    let lastAppliedWidth = -1;
    let iteration = 0;
    const MAX_ITERATIONS = 6;

    const measure = () => {
      if (disposed) return;
      el.style.width = "";
      el.getBoundingClientRect(); // force reflow
      const range = document.createRange();
      range.selectNodeContents(el);
      // jsdom no-op — Range.getClientRects isn't implemented there.
      if (typeof range.getClientRects !== "function") return;
      // Range misses inline replaced elements (svg/img) and inline-flex
      // anchor gaps; querySelector fills both. Group by vertical center
      // with half-line fuzz so a flex-centered icon rejoins its text line.
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
      centersSorted.forEach((item) => {
        const last = lines[lines.length - 1];
        if (last && Math.abs(item.center - last.center) < fuzz) {
          if (item.left < last.left) last.left = item.left;
          if (item.right > last.right) last.right = item.right;
        } else {
          lines.push({ ...item });
        }
      });
      let widest = 0;
      lines.forEach((line) => {
        const lineWidth = line.right - line.left;
        if (lineWidth > widest) widest = lineWidth;
      });
      if (widest <= 0) return;
      const next = Math.ceil(widest);
      // Restore the previous (narrower) width and stop once shrinking flattens.
      if (next >= lastAppliedWidth && lastAppliedWidth !== -1) {
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

    // Web-font load can reflow bold/italic runs; re-run the loop after.
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

  return (
    <div ref={ref} style={{ display: "inline-block", textWrap: "balance" }}>
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
