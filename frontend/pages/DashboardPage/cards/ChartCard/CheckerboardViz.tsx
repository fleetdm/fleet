import React, { useCallback, useMemo, useRef, useState } from "react";
import { format, parseISO } from "date-fns";

import { IFormattedDataPoint } from "./types";

const baseClass = "checkerboard-viz";

// Returns a CSS class suffix for the color level (0-5)
const getColorLevel = (percentage: number): number => {
  if (percentage === 0) return 0;
  if (percentage <= 25) return 1;
  if (percentage <= 50) return 2;
  if (percentage <= 75) return 3;
  return 4;
};

const formatHourLabel = (hourVal: number): string => {
  if (hourVal === 0) return "12am";
  if (hourVal < 12) return `${hourVal}am`;
  if (hourVal === 12) return "12pm";
  return `${hourVal - 12}pm`;
};

// Font Awesome "moon" (classic solid) — crescent moon
// eslint-disable-next-line react/prop-types
const MoonIcon = ({ size = 16, color = "#6C7A89" }) => (
  <svg
    viewBox="0 0 384 512"
    width={size}
    height={size}
    fill={color}
    aria-hidden="true"
  >
    <path d="M223.5 32C100 32 0 132.3 0 256s100 224 223.5 224c60.6 0 115.5-24.2 155.8-63.4 5-4.9 6.3-12.5 3.1-18.7s-10.1-9.7-17-8.5c-9.8 1.7-19.8 2.6-30.1 2.6-96.9 0-175.5-78.8-175.5-176 0-65.8 36-123.1 89.3-153.3 6.1-3.5 9.2-10.5 7.7-17.3s-7.3-11.9-14.3-12.5c-6.3-.5-12.6-.8-19-.8z" />
  </svg>
);

// Font Awesome "sun-bright" (classic solid) — sun with rays
// eslint-disable-next-line react/prop-types
const SunIcon = ({ size = 16, color = "#6C7A89" }) => (
  <svg
    viewBox="0 0 512 512"
    width={size}
    height={size}
    fill={color}
    aria-hidden="true"
  >
    <path d="M256 0c-13.3 0-24 10.7-24 24v56c0 13.3 10.7 24 24 24s24-10.7 24-24V24c0-13.3-10.7-24-24-24zm0 408c-13.3 0-24 10.7-24 24v56c0 13.3 10.7 24 24 24s24-10.7 24-24v-56c0-13.3-10.7-24-24-24zM488 232h-56c-13.3 0-24 10.7-24 24s10.7 24 24 24h56c13.3 0 24-10.7 24-24s-10.7-24-24-24zM80 232H24c-13.3 0-24 10.7-24 24s10.7 24 24 24h56c13.3 0 24-10.7 24-24s-10.7-24-24-24zm340.5-91.2-39.6 39.6c-9.4 9.4-9.4 24.6 0 33.9 9.4 9.4 24.6 9.4 33.9 0l39.6-39.6c9.4-9.4 9.4-24.6 0-33.9s-24.6-9.4-33.9 0zM97.1 337.7l-39.6 39.6c-9.4 9.4-9.4 24.6 0 33.9s24.6 9.4 33.9 0l39.6-39.6c-9.4-9.4-24.6-9.4-33.9 0-9.4-9.4-9.4-24.6 0-33.9zm316.8 73.5c9.4-9.4 9.4-24.6 0-33.9l-39.6-39.6c-9.4-9.4-24.6-9.4-33.9 0s-9.4 24.6 0 33.9l39.6 39.6c9.4 9.4 24.6 9.4 33.9 0zM131 180.3l-39.6-39.6c-9.4-9.4-24.6-9.4-33.9 0s-9.4 24.6 0 33.9l39.6 39.6c9.4 9.4 24.6 9.4 33.9 0s9.4-24.6 0-33.9zM256 152c-57.4 0-104 46.6-104 104s46.6 104 104 104 104-46.6 104-104-46.6-104-104-104z" />
  </svg>
);

interface ICellData {
  dayIndex: number;
  hourRow: number;
  percentage: number;
  dayLabel: string;
  hourLabel: string;
}

interface ICheckerboardVizProps {
  data: IFormattedDataPoint[];
  selectedDays: number;
  isPercentage: boolean;
}

const CELL_W = 16;
const CELL_H = 18;
const CELL_GAP = 2;
const Y_AXIS_WIDTH = 40; // space for y-axis icons on the left
// When cards stack to full-width (below $break-md), the container gets wider
// than this threshold and we scale cells up by WIDE_MULTIPLIER.
const WIDE_THRESHOLD = 700;
const WIDE_MULTIPLIER = 1.5;

// Determine which y-axis section a row belongs to based on the hour it represents
type TimeOfDay = "night-top" | "day" | "night-bottom";

const getTimeOfDay = (row: number, hoursPerSlot: number): TimeOfDay => {
  const hourStart = row * hoursPerSlot;
  if (hourStart < 6) return "night-top";
  if (hourStart < 18) return "day";
  return "night-bottom";
};

const CheckerboardViz = ({
  data,
  selectedDays,
}: ICheckerboardVizProps): JSX.Element => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [isWide, setIsWide] = useState(false);
  const [hoveredCell, setHoveredCell] = useState<ICellData | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });

  const measuredRef = useCallback((node: HTMLDivElement | null) => {
    if (node) {
      (containerRef as React.MutableRefObject<HTMLDivElement>).current = node;
      const observer = new ResizeObserver((entries) => {
        const entry = entries[0];
        if (entry) {
          setIsWide(entry.contentRect.width >= WIDE_THRESHOLD);
        }
      });
      observer.observe(node);
      setIsWide(node.getBoundingClientRect().width >= WIDE_THRESHOLD);
      return () => observer.disconnect();
    }
    return undefined;
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
    if (is24h) {
      // 24h: flat sequence of data points as columns, no day grouping
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

    const dayMap = new Map<string, Map<number, IFormattedDataPoint>>();
    const dayOrder: string[] = [];

    data.forEach((point) => {
      const date = parseISO(point.timestamp);
      const dayKey = format(date, "yyyy-MM-dd");
      const hour = date.getHours();
      const slot = Math.floor(hour / hoursPerSlot);

      if (!dayMap.has(dayKey)) {
        dayMap.set(dayKey, new Map());
        dayOrder.push(dayKey);
      }
      const hourMap = dayMap.get(dayKey);
      if (hourMap) {
        const existing = hourMap.get(slot);
        if (!existing || point.percentage > existing.percentage) {
          hourMap.set(slot, point);
        }
      }
    });

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

  // Compute y-axis icon positions (only for multi-day views)
  const yAxisSections = useMemo(() => {
    if (is24h) return [];

    const sections: {
      type: TimeOfDay;
      startRow: number;
      endRow: number;
    }[] = [];
    let currentType = getTimeOfDay(0, hoursPerSlot);
    let startRow = 0;

    for (let row = 1; row < hourRows; row += 1) {
      const tod = getTimeOfDay(row, hoursPerSlot);
      if (tod !== currentType) {
        sections.push({ type: currentType, startRow, endRow: row - 1 });
        currentType = tod;
        startRow = row;
      }
    }
    sections.push({
      type: currentType,
      startRow,
      endRow: hourRows - 1,
    });

    return sections;
  }, [is24h, hoursPerSlot, hourRows]);

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
      setTooltipPos({
        x: rect.left - containerRect.left + cellW / 2,
        y: rect.top - containerRect.top - 8,
      });
    }
  };

  const handleMouseLeave = () => {
    setHoveredCell(null);
  };

  const showYAxis = !is24h;
  const leftMargin = showYAxis ? Y_AXIS_WIDTH : 0;
  const iconSize = 16 * scale;

  return (
    <div
      className={baseClass}
      ref={measuredRef}
      style={{ position: "relative" }}
    >
      <div className={`${baseClass}__scroll-wrapper`}>
        <div className={`${baseClass}__grid-area`}>
          {/* Y-axis icons */}
          {showYAxis && (
            <div
              className={`${baseClass}__y-axis`}
              style={{ width: leftMargin, height: gridHeight }}
            >
              {yAxisSections.map((section) => {
                const topPx = section.startRow * (cellH + CELL_GAP);
                const rowCount = section.endRow - section.startRow + 1;
                const sectionHeight =
                  rowCount * cellH + (rowCount - 1) * CELL_GAP;
                return (
                  <div
                    key={`${section.type}-${section.startRow}`}
                    className={`${baseClass}__y-axis-icon`}
                    style={{
                      position: "absolute",
                      top: topPx,
                      left: 0,
                      width: leftMargin,
                      height: sectionHeight,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                    }}
                  >
                    {section.type === "day" ? (
                      <SunIcon size={iconSize} color="#9FAAB5" />
                    ) : (
                      <MoonIcon size={iconSize} color="#9FAAB5" />
                    )}
                  </div>
                );
              })}
            </div>
          )}

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
                  onMouseEnter={(e) => handleMouseEnter(cell, e)}
                  onMouseLeave={handleMouseLeave}
                />
              );
            })}
          </svg>
        </div>

        {/* X-axis date labels */}
        {!is24h && dayLabels.length >= 2 && (
          <div
            className={`${baseClass}__x-axis`}
            style={{ marginLeft: leftMargin }}
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

      {hoveredCell && (
        <div
          className="chart-card__tooltip"
          style={{
            position: "absolute",
            left: tooltipPos.x,
            top: tooltipPos.y,
            transform: "translate(-50%, -100%)",
            pointerEvents: "none",
          }}
        >
          <div className="chart-card__tooltip-label">
            {hoveredCell.dayLabel}, {hoveredCell.hourLabel}
          </div>
          <div className="chart-card__tooltip-value">
            {hoveredCell.percentage}% of hosts
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
