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

const HOUR_ROWS = 12; // 0, 2, 4, ..., 22
const CELL_GAP = 2;
const AXIS_HEIGHT = 20; // space for x-axis labels at bottom
const CHART_HEIGHT = 260; // target height for the grid area

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

  const { grid, dayLabels } = useMemo(() => {
    const dayMap = new Map<string, Map<number, IFormattedDataPoint>>();
    const dayOrder: string[] = [];

    data.forEach((point) => {
      const date = parseISO(point.timestamp);
      const dayKey = format(date, "yyyy-MM-dd");
      const hour = date.getHours();
      const slot = Math.floor(hour / 2);

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

      for (let row = 0; row < HOUR_ROWS; row += 1) {
        const point = hourMap?.get(row);
        const hourVal = row * 2;
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
  }, [data]);

  const numDays = dayLabels.length || 1;

  // Cell width fills the container, cell height fills the target chart height
  const cellW = containerWidth
    ? (containerWidth - CELL_GAP * (numDays - 1)) / numDays
    : 0;
  const cellH = (CHART_HEIGHT - CELL_GAP * (HOUR_ROWS - 1)) / HOUR_ROWS;
  const gridHeight = cellH * HOUR_ROWS + CELL_GAP * (HOUR_ROWS - 1);
  const svgHeight = gridHeight + AXIS_HEIGHT;

  // Show ~6 x-axis labels
  const tickInterval = Math.max(1, Math.floor(numDays / 6));

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
          {grid.map((cell) => (
            <rect
              key={`${cell.dayIndex}-${cell.hourRow}`}
              x={cell.dayIndex * (cellW + CELL_GAP)}
              y={cell.hourRow * (cellH + CELL_GAP)}
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
          ))}
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
