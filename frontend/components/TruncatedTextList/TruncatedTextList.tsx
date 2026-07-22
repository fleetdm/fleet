import React, { useLayoutEffect, useRef, useState } from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "truncated-text-list";

interface ITruncatedTextListProps {
  items: string[];
  /** Inserted between items in both the visible row and the tooltip. */
  separator?: string;
  /** Tooltip placement. */
  tooltipPosition?: "top" | "bottom" | "left" | "right";
  /** Approximate character budget for the first label when even it doesn't
   * fit alongside the "+N more" pill. The first label is truncated to this
   * many chars and gets a trailing ellipsis. Default 30. */
  truncatedFirstMaxChars?: number;
  /** When provided, the whole visible row renders as `Button variant="link"`
   * (CustomLink-style animated underline) and calls this handler on click. */
  onClick?: () => void;
  className?: string;
}

const truncateString = (s: string, max: number) =>
  s.length > max ? `${s.slice(0, max).trimEnd()}...` : s;

const renderItemsList = (list: string[]) => (
  <>
    {list.map((name, i) => (
      <React.Fragment key={name}>
        {name}
        {i < list.length - 1 && <br />}
      </React.Fragment>
    ))}
  </>
);

interface IRenderVisibleRowParams {
  visibleCount: number;
  visible: string[];
  hidden: string[];
  items: string[];
  separator: string;
  tooltipPosition: "top" | "bottom" | "left" | "right";
  truncatedFirstMaxChars: number;
  onClick?: () => void;
}

const renderVisibleRow = ({
  visibleCount,
  visible,
  hidden,
  items,
  separator,
  tooltipPosition,
  truncatedFirstMaxChars,
  onClick,
}: IRenderVisibleRowParams) => {
  const truncatedFirst = truncateString(items[0] ?? "", truncatedFirstMaxChars);
  const firstWasTruncated = truncatedFirst !== (items[0] ?? "");

  const truncatedFirstContent = (
    <>
      {firstWasTruncated ? (
        <TooltipWrapper
          tipContent={items[0]}
          showArrow
          underline={false}
          position={tooltipPosition}
          tipOffset={8}
          fixedPositionStrategy
        >
          <span>{truncatedFirst}</span>
        </TooltipWrapper>
      ) : (
        <span>{truncatedFirst}</span>
      )}
      {items.length > 1 && (
        <>
          {separator}
          <TooltipWrapper
            tipContent={renderItemsList(items.slice(1))}
            showArrow
            underline={false}
            position={tooltipPosition}
            tipOffset={8}
            fixedPositionStrategy
          >
            <span className={`${baseClass}__more`}>
              +{items.length - 1} more
            </span>
          </TooltipWrapper>
        </>
      )}
    </>
  );

  const standardContent = (
    <>
      {visible.join(separator)}
      {hidden.length > 0 && (
        <>
          {visible.length > 0 ? separator : ""}
          <TooltipWrapper
            tipContent={renderItemsList(hidden)}
            showArrow
            underline={false}
            position={tooltipPosition}
            tipOffset={8}
            fixedPositionStrategy
          >
            <span className={`${baseClass}__more`}>+{hidden.length} more</span>
          </TooltipWrapper>
        </>
      )}
    </>
  );

  const isTruncatedFirst = visibleCount === 0;
  const content = isTruncatedFirst ? truncatedFirstContent : standardContent;
  const rowClass = classnames(`${baseClass}__visible`, {
    [`${baseClass}__visible--truncated`]: isTruncatedFirst,
  });

  if (onClick) {
    return (
      <Button variant="link" className={rowClass} onClick={onClick}>
        <span>{content}</span>
      </Button>
    );
  }

  return <span className={rowClass}>{content}</span>;
};

const TruncatedTextList = ({
  items,
  separator = ", ",
  tooltipPosition = "top",
  truncatedFirstMaxChars = 30,
  onClick,
  className,
}: ITruncatedTextListProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef<(HTMLSpanElement | null)[]>([]);
  const moreRef = useRef<HTMLSpanElement>(null);
  const [visibleCount, setVisibleCount] = useState(items.length);

  useLayoutEffect(() => {
    const measure = () => {
      if (!containerRef.current) return;
      // `getBoundingClientRect().width` gives subpixel precision —
      // `clientWidth`/`offsetWidth` round to integers, which lets a row that
      // sums to (say) container_width + 0.7px read as "fits." Plus a small
      // buffer for the layout discrepancy between the measure layer (each
      // item in its own `<span>`) and the visible row (items joined into a
      // single string), which can differ by a few pixels from inline
      // boundary kerning. Same spirit as the `max-width: 101%` subpixel
      // trick in `TooltipTruncatedTextCell`.
      const BOUNDARY_BUFFER_PX = 16;
      const containerWidth =
        containerRef.current.getBoundingClientRect().width - BOUNDARY_BUFFER_PX;
      const moreWidth = moreRef.current?.getBoundingClientRect().width ?? 0;

      const widths = itemRefs.current.map(
        (el) => el?.getBoundingClientRect().width ?? 0
      );
      const totalWidth = widths.reduce((sum, w) => sum + w, 0);

      // Everything fits — no "+N more" needed.
      if (totalWidth <= containerWidth) {
        setVisibleCount(items.length);
        return;
      }

      // Some items must be hidden — reserve room for the "+N more" pill.
      let used = 0;
      let count = 0;
      for (let i = 0; i < widths.length; i += 1) {
        if (used + widths[i] + moreWidth > containerWidth) break;
        used += widths[i];
        count += 1;
      }
      setVisibleCount(count);
    };

    measure();

    if (!containerRef.current) return undefined;
    const observer = new ResizeObserver(measure);
    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, [items]);

  if (items.length === 0) return null;

  const visible = items.slice(0, visibleCount);
  const hidden = items.slice(visibleCount);

  return (
    <div ref={containerRef} className={classnames(baseClass, className)}>
      {/* Hidden measurement layer — same font/size as the visible row */}
      <div className={`${baseClass}__measure`} aria-hidden>
        {items.map((item, i) => (
          <span
            // eslint-disable-next-line react/no-array-index-key
            key={`measure-${item}-${i}`}
            ref={(el) => {
              itemRefs.current[i] = el;
            }}
          >
            {i > 0 ? separator : ""}
            {item}
          </span>
        ))}
        <span ref={moreRef}>
          {separator}+{items.length} more
        </span>
      </div>

      {/* Visible row */}
      {renderVisibleRow({
        visibleCount,
        visible,
        hidden,
        items,
        separator,
        tooltipPosition,
        truncatedFirstMaxChars,
        onClick,
      })}
    </div>
  );
};

export default TruncatedTextList;
