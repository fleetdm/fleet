import React, { useMemo, useRef, useState, useCallback } from "react";
import { format, parseISO } from "date-fns";

import { IFormattedDataPoint } from "./types";

const baseClass = "checkerboard-viz";

// Returns a CSS class suffix for the color level (0-5)
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
  percentage: number;
  dayLabel: string;
  hourLabel: string;
}

interface ICheckerboardVizProps {
  data: IFormattedDataPoint[];
  selectedDays: number;
  isPercentage: boolean;
}

const CELL_GAP = 2;
const AXIS_HEIGHT = 20; // space for x-axis labels at bottom
const CHART_HEIGHT = 260; // target height for the grid area (non-30-day)
const CHART_HEIGHT_30D = 130; // half height for 30-day view

const CheckerboardViz = ({
  data,
  selectedDays,
  isPercentage,
}: ICheckerboardVizProps): JSX.Element => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(0);
  const [hoveredCell, setHoveredCell] = useState<ICellData | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });

  const measuredRef = useCallback((node: HTMLDivElement | null) => {
    if (node) {
      // Store ref and measure
      (containerRef as React.MutableRefObject<HTMLDivElement>).current = node;
      const observer = new ResizeObserver((entries) => {
        const entry = entries[0];
        if (entry) {
          setContainerWidth(entry.contentRect.width);
        }
      });
      observer.observe(node);
      // Initial measurement
      setContainerWidth(node.getBoundingClientRect().width);
      return () => observer.disconnect();
    }
    return undefined;
  }, []);

  // Hours per slot: 4 for 30-day, 2 for 7/14-day, 1 for 24-hour
  const is24h = selectedDays === 1;
  let hoursPerSlot = 1;
  if (selectedDays === 30) {
    hoursPerSlot = 4;
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

  const cellW = containerWidth
    ? (containerWidth - CELL_GAP * (numCols - 1)) / numCols
    : 0;

  let chartHeight: number;
  if (selectedDays === 30) {
    chartHeight = CHART_HEIGHT_30D;
  } else if (is24h) {
    chartHeight = 40; // single row
  } else {
    chartHeight = CHART_HEIGHT;
  }

  const cellH = (chartHeight - CELL_GAP * (numRows - 1)) / numRows;
  const gridHeight = cellH * numRows + CELL_GAP * (numRows - 1);
  const svgHeight = gridHeight + AXIS_HEIGHT;

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

  return (
    <div
      className={baseClass}
      ref={measuredRef}
      style={{ position: "relative" }}
    >
      {cellW > 0 && (
        <svg width="100%" height={svgHeight}>
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
      )}
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
