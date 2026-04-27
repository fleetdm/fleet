import React, { useEffect, useMemo, useRef, useState } from "react";
import classnames from "classnames";
import { format, parseISO } from "date-fns";

import { ChartTheme, IFormattedDataPoint, TooltipFormatter } from "./types";

const baseClass = "checkerboard-viz";

// Returns a CSS class suffix for the color level (0-5). Buckets match the
// six legend swatches declared in _styles.scss (level-0 is the no-data swatch).
const getColorLevel = (percentage: number): number => {
  if (percentage === 0) return 0;
  if (percentage <= 20) return 1;
  if (percentage <= 40) return 2;
  if (percentage <= 60) return 3;
  if (percentage <= 80) return 4;
  return 5;
};

const formatHourLabel = (hourVal: number): string => {
  if (hourVal === 0) return "12am";
  if (hourVal < 12) return `${hourVal}am`;
  if (hourVal === 12) return "12pm";
  return `${hourVal - 12}pm`;
};

interface ICellData {
  dayIndex: number;
  hourRow: number;
  value: number;
  total?: number;
  percentage: number;
  dayLabel: string;
  hourLabel: string;
}

interface ICheckerboardVizProps {
  data: IFormattedDataPoint[];
  selectedDays: number;
  theme?: ChartTheme;
  tooltipFormatter?: TooltipFormatter;
}

// These are calculated at a chart width of 580px and columns.
const CELL_W = 16.75;
const CELL_H = 19;
const CELL_GAP = 2;
const Y_AXIS_WIDTH = 40; // space for y-axis labels on the left
// When cards stack to full-width (below $break-md), the container gets wider
// than this threshold and we scale cells up by WIDE_MULTIPLIER.
const WIDE_THRESHOLD = 700;
const WIDE_MULTIPLIER = 1.5;

const CheckerboardViz = ({
  data,
  selectedDays,
  theme = "green",
  tooltipFormatter,
}: ICheckerboardVizProps): JSX.Element => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [isWide, setIsWide] = useState(false);
  const [hoveredCell, setHoveredCell] = useState<ICellData | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });
  const [tooltipAlign, setTooltipAlign] = useState<"left" | "center" | "right">(
    "center"
  );

  useEffect(() => {
    const node = containerRef.current;
    if (!node) return undefined;
    setIsWide(node.getBoundingClientRect().width >= WIDE_THRESHOLD);
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (entry) {
        setIsWide(entry.contentRect.width >= WIDE_THRESHOLD);
      }
    });
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  // Hours per slot: 3 for 30-day, 2 for 7/14-day, 1 for 24-hour
  const is24h = selectedDays === 1;
  let hoursPerSlot = 1;
  if (selectedDays === 30) {
    hoursPerSlot = 3;
  } else if (selectedDays >= 7) {
    hoursPerSlot = 2;
  }
  const hourRows = 24 / hoursPerSlot;

  const { grid, dayLabels } = useMemo(() => {
    // 24h view: each incoming data point becomes a single column in a
    // one-row strip. No day grouping, no slot aggregation — the backend has
    // already produced one point per hour and we render them in order.
    if (is24h) {
      const cells: ICellData[] = data.map((point, i) => {
        const date = parseISO(point.timestamp);
        return {
          dayIndex: 0,
          hourRow: i,
          percentage: point.percentage,
          dayLabel: format(date, "MMM d"),
          hourLabel: formatHourLabel(date.getHours()),
        };
      });
      return { grid: cells, dayLabels: ["today"] };
    }

    // Multi-day view: build a day × time-slot grid.
    //
    // `dayMap` is keyed by calendar day (yyyy-MM-dd) and each day holds a
    // map of slot-index → data point. `dayOrder` preserves first-seen order
    // so columns render chronologically regardless of Map iteration quirks.
    const dayMap = new Map<string, Map<number, IFormattedDataPoint>>();
    const dayOrder: string[] = [];

    data.forEach((point) => {
      const date = parseISO(point.timestamp);
      const dayKey = format(date, "yyyy-MM-dd");
      const hour = date.getHours();
      // `hoursPerSlot` > 1 for wider windows (2 for 7/14-day, 3 for 30-day),
      // so multiple hourly points collapse into the same slot row.
      const slot = Math.floor(hour / hoursPerSlot);

      if (!dayMap.has(dayKey)) {
        dayMap.set(dayKey, new Map());
        dayOrder.push(dayKey);
      }
      const hourMap = dayMap.get(dayKey);
      if (hourMap) {
        // When several hourly points land in the same slot, keep the max
        // percentage. This biases visualizations toward "hosts had data
        // at some point during this window".
        const existing = hourMap.get(slot);
        if (!existing || point.percentage > existing.percentage) {
          hourMap.set(slot, point);
        }
      }
    });

    // Emit one cell per (day, slot) pair in a fixed grid. Days with no data
    // for a given slot still get a cell with percentage 0 so the grid stays
    // rectangular and the color ramp renders the "no data" swatch.
    const labels: string[] = [];
    const cells: ICellData[] = [];

    dayOrder.forEach((dayKey, dayIndex) => {
      const date = parseISO(dayKey);
      labels.push(format(date, "MMM d"));
      const hourMap = dayMap.get(dayKey);

      for (let row = 0; row < hourRows; row += 1) {
        const point = hourMap?.get(row);
        const hourVal = row * hoursPerSlot;
        cells.push({
          dayIndex,
          hourRow: row,
          percentage: point?.percentage ?? 0,
          value: point?.value ?? 0,
          total: point?.total,
          dayLabel: format(date, "MMM d"),
          hourLabel: formatHourLabel(hourVal),
        });
      }
    });

    return { grid: cells, dayLabels: labels };
  }, [data, hoursPerSlot, hourRows, is24h]);

  const numDays = dayLabels.length || 1;

  // For 24h: hours are columns, single row. Otherwise: days are columns, hours are rows.
  const numCols = is24h ? hourRows : numDays;
  const numRows = is24h ? 1 : hourRows;

  const scale = isWide ? WIDE_MULTIPLIER : 1;
  const cellW = CELL_W * scale;
  const cellH = CELL_H * scale;
  const gridWidth = cellW * numCols + CELL_GAP * (numCols - 1);
  const gridHeight = cellH * numRows + CELL_GAP * (numRows - 1);

  // Compute x-axis date labels: start, middle, end
  const xAxisDates = useMemo(() => {
    if (dayLabels.length < 2) return { start: "", middle: "", end: "" };
    const midIndex = Math.floor(dayLabels.length / 2);
    return {
      start: dayLabels[0],
      middle: dayLabels[midIndex],
      end: dayLabels[dayLabels.length - 1],
    };
  }, [dayLabels]);

  const handleMouseEnter = (cell: ICellData, e: React.MouseEvent) => {
    setHoveredCell(cell);
    const rect = (e.target as SVGElement).getBoundingClientRect();
    const containerRect = containerRef.current?.getBoundingClientRect();
    if (containerRect) {
      const cellCenterX = rect.left - containerRect.left + cellW / 2;
      setTooltipPos({
        x: cellCenterX,
        y: rect.top - containerRect.top - 8,
      });
      // Near either edge, the default centered tooltip would overflow or
      // word-wrap. Flip the anchor so the tooltip grows inward instead.
      const EDGE_ZONE = 100;
      if (cellCenterX > containerRect.width - EDGE_ZONE) {
        setTooltipAlign("right");
      } else if (cellCenterX < EDGE_ZONE) {
        setTooltipAlign("left");
      } else {
        setTooltipAlign("center");
      }
    }
  };

  const handleMouseLeave = () => {
    setHoveredCell(null);
  };

  const showYAxis = !is24h;
  const leftMargin = showYAxis ? Y_AXIS_WIDTH : 0;

  return (
    <div
      className={classnames(baseClass, `${baseClass}--theme-${theme}`)}
      ref={containerRef}
    >
      <div className={`${baseClass}__chart-row`}>
        {/* Y-axis 6am/6pm labels. Kept outside the scroll-wrapper so they stay
            pinned during horizontal scroll and label text can overflow
            leftward without being clipped. */}
        {showYAxis && (
          <div
            className={`${baseClass}__y-axis`}
            style={{ width: leftMargin, height: gridHeight }}
          >
            {[
              { hour: 6, label: "6am" },
              { hour: 18, label: "6pm" },
            ].map(({ hour, label }) => {
              const row = hour / hoursPerSlot;
              // Position label centered vertically on the row that represents
              // this hour.
              const topPx = row * (cellH + CELL_GAP) + cellH / 2;
              return (
                <div
                  key={label}
                  className={`${baseClass}__y-axis-label`}
                  style={{ top: topPx }}
                >
                  {label}
                </div>
              );
            })}
          </div>
        )}

        <div className={`${baseClass}__scroll-wrapper`}>
          {/* Grid cells */}
          <svg width={gridWidth} height={gridHeight}>
            {grid.map((cell) => {
              const col = is24h ? cell.hourRow : cell.dayIndex;
              const row = is24h ? 0 : cell.hourRow;
              return (
                <rect
                  key={`${cell.dayIndex}-${cell.hourRow}`}
                  x={col * (cellW + CELL_GAP)}
                  y={row * (cellH + CELL_GAP)}
                  width={cellW}
                  height={cellH}
                  rx={3}
                  ry={3}
                  className={`${baseClass}__cell ${baseClass}__cell--level-${getColorLevel(
                    cell.percentage
                  )}`}
                  role="img"
                  aria-label={`${cell.dayLabel}, ${cell.hourLabel}: ${cell.percentage}% of hosts online`}
                  onMouseEnter={(e) => handleMouseEnter(cell, e)}
                  onMouseLeave={handleMouseLeave}
                />
              );
            })}
          </svg>

          {/* X-axis date labels */}
          {!is24h && dayLabels.length >= 2 && (
            <div
              className={`${baseClass}__x-axis`}
              style={{ width: gridWidth }}
            >
              <span className={`${baseClass}__x-axis-label`}>
                {xAxisDates.start}
              </span>
              <span className={`${baseClass}__x-axis-label`}>
                {xAxisDates.middle}
              </span>
              <span
                className={`${baseClass}__x-axis-label ${baseClass}__x-axis-label--end`}
              >
                {xAxisDates.end}
              </span>
            </div>
          )}
        </div>
      </div>

      {hoveredCell && (
        <div
          className={`chart-card__tooltip ${baseClass}__floating-tooltip ${baseClass}__floating-tooltip--align-${tooltipAlign}`}
          style={{ left: tooltipPos.x, top: tooltipPos.y }}
        >
          <div className="chart-card__tooltip-label">
            {hoveredCell.dayLabel}, {hoveredCell.hourLabel}
          </div>
          <div className="chart-card__tooltip-value">
            {tooltipFormatter
              ? tooltipFormatter({
                  value: hoveredCell.value,
                  percentage: hoveredCell.percentage,
                  total: hoveredCell.total,
                })
              : `${hoveredCell.percentage}% of hosts`}
          </div>
        </div>
      )}
      <div className={`${baseClass}__legend`}>
        <span className={`${baseClass}__legend-label`}>No data</span>
        <span
          className={`${baseClass}__legend-swatch ${baseClass}__cell--level-0`}
        />
        <span className={`${baseClass}__legend-label`}>Less</span>
        {[1, 2, 3, 4, 5].map((level) => (
          <span
            key={level}
            className={`${baseClass}__legend-swatch ${baseClass}__cell--level-${level}`}
          />
        ))}
        <span className={`${baseClass}__legend-label`}>More</span>
      </div>
    </div>
  );
};

export default CheckerboardViz;
