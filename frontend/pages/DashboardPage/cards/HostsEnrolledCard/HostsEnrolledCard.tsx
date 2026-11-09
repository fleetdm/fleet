import classnames from "classnames";
import React, { useEffect, useRef, useState } from "react";
import { InjectedRouter } from "react-router";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
  Cell,
  Tooltip,
} from "recharts";

import { getBuiltinPlatformLabelId, ILabelSummary } from "interfaces/label";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

const baseClass = "hosts-enrolled-card";

// Use design-system color tokens via CSS custom properties so recharts picks
// up the themed values for the SVG fills.
const BAR_COLOR = "var(--core-fleet-green)";
const BAR_HOVER_COLOR = "var(--core-fleet-green-over)";

// Mirror the sibling checkerboard's cell-grid height so the bar plot lines up
// with the cells next to it. Formula, constants, and 30-day column count come
// from CheckerboardViz.tsx; keep them in sync if they change there. The
// sibling's y-axis gutter (Y_AXIS_WIDTH, 40) is subtracted from our container
// width to recover its scroll-wrapper width, since both cards are equal-width
// columns in __charts-row.
const CELL_W = 16.75;
const CELL_H = 19;
const CELL_MAX_SCALE = 2;
const CELL_GAP = 2;
const CHECKERBOARD_Y_AXIS_WIDTH = 40;
const CHECKERBOARD_NUM_COLS = 30;
const CHECKERBOARD_NUM_ROWS = 8;

// ChromeOS is the widest y-axis label and clips at 12px medium weight without
// the wider gutter. Width concern only, unrelated to cell-grid alignment.
const WIDE_YAXIS_THRESHOLD = 700;

export interface IHostPlatformCounts {
  darwin: number;
  windows: number;
  linux: number;
  chrome: number;
  ios: number;
  ipados: number;
  android: number;
}

type PlatformKey = keyof IHostPlatformCounts;

interface IHostsEnrolledCardProps {
  counts: IHostPlatformCounts;
  // Total enrolled hosts across the whole fleet, used as the denominator for
  // each platform's share. This is the authoritative total (unaffected by the
  // dashboard's platform filter), not the sum of the bars below — some hosts
  // may be on platforms not broken out here.
  totalHostCount: number;
  builtInLabels?: ILabelSummary[];
  currentTeamId?: number;
  router: InjectedRouter;
}

interface IPlatformDatum {
  label: string;
  count: number;
  platform: PlatformKey;
  percent: number;
}

// Make a map of platform to label for use in linking to the correct hosts list
// when clicked.
const PLATFORM_ROWS: { platform: PlatformKey; label: string }[] = [
  { platform: "darwin", label: "macOS" },
  { platform: "windows", label: "Windows" },
  { platform: "linux", label: "Linux" },
  { platform: "chrome", label: "ChromeOS" },
  { platform: "ios", label: "iOS" },
  { platform: "ipados", label: "iPadOS" },
  { platform: "android", label: "Android" },
];

const formatTick = (value: number): string => {
  if (value >= 1000) {
    const k = value / 1000;
    return Number.isInteger(k) ? `${k}k` : `${k.toFixed(1)}k`;
  }
  return `${value}`;
};

// Format a platform's share of all enrolled hosts. A nonzero count that rounds
// to 0.0% (e.g. a handful of hosts in a fleet of tens of thousands) shows as
// "<0.1%" rather than misleadingly reading "0.0%".
export const formatPercent = (count: number, percent: number): string => {
  if (count > 0 && percent < 0.1) return "<0.1%";
  return `${percent.toFixed(1)}%`;
};

interface ITooltipProps {
  active?: boolean;
  payload?: { payload: IPlatformDatum }[];
}

const HostsEnrolledTooltip = ({
  active,
  payload,
}: ITooltipProps): JSX.Element | null => {
  if (!active || !payload?.length) return null;
  const datum = payload[0].payload;
  const percentLabel = formatPercent(datum.count, datum.percent);
  return (
    <div className={`${baseClass}__tooltip`}>
      <div className={`${baseClass}__tooltip-label`}>{datum.label}</div>
      <div className={`${baseClass}__tooltip-value`}>
        {datum.count.toLocaleString()} hosts
      </div>
      <div className={`${baseClass}__tooltip-share`}>
        {percentLabel} of fleet
      </div>
    </div>
  );
};

interface IYAxisTickProps {
  x?: number;
  y?: number;
  payload?: { value: string; index: number };
  // recharts merges its own "recharts-cartesian-axis-tick-value" class in via
  // cloneElement; forward it so recharts' internal tick measurement still works.
  className?: string;
  fontSize: number;
  isClickable: (index: number) => boolean;
  onLabelClick: (index: number) => void;
}

const ClickableYAxisTick = ({
  x = 0,
  y = 0,
  payload,
  className,
  fontSize,
  isClickable,
  onLabelClick,
}: IYAxisTickProps): JSX.Element => {
  if (!payload) return <g />;
  const clickable = isClickable(payload.index);

  // Make clickable platform labels real, keyboard-operable controls: focusable
  // via Tab (tabIndex), announced as buttons, and activatable with Enter/Space
  // in addition to a mouse click. They navigate programmatically (no href), so
  // button semantics fit better than a link. See #48214.
  const handleKeyDown = (event: React.KeyboardEvent<SVGTextElement>) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onLabelClick(payload.index);
    }
  };

  return (
    <g transform={`translate(${x},${y})`}>
      <text
        x={0}
        y={0}
        dy={4}
        textAnchor="end"
        fontSize={fontSize}
        fontWeight="normal"
        className={classnames(className, {
          [`${baseClass}__tick--clickable`]: clickable,
        })}
        role={clickable ? "button" : undefined}
        tabIndex={clickable ? 0 : undefined}
        aria-label={clickable ? `${payload.value} hosts` : undefined}
        onClick={clickable ? () => onLabelClick(payload.index) : undefined}
        onKeyDown={clickable ? handleKeyDown : undefined}
      >
        {payload.value}
      </text>
    </g>
  );
};

const HostsEnrolledCard = ({
  counts,
  totalHostCount,
  builtInLabels,
  currentTeamId,
  router,
}: IHostsEnrolledCardProps): JSX.Element => {
  const data: IPlatformDatum[] = PLATFORM_ROWS.map(({ platform, label }) => ({
    platform,
    label,
    count: counts[platform],
    percent: totalHostCount ? (counts[platform] / totalHostCount) * 100 : 0,
  }));

  const navigateToPlatform = (platform: PlatformKey, count: number) => {
    if (!count) return;
    const labelId = getBuiltinPlatformLabelId(builtInLabels, platform);
    if (labelId === undefined) return;
    router.push(
      getPathWithQueryParams(PATHS.MANAGE_HOSTS_LABEL(labelId), {
        fleet_id: currentTeamId,
        // the chart doesn't count hosts pending MDM enrollment
        status: "enrolled",
      })
    );
  };

  const handleBarClick = (datum: IPlatformDatum) => {
    navigateToPlatform(datum.platform, datum.count);
  };

  const handleTickClick = (index: number) => {
    const datum = data[index];
    if (datum) navigateToPlatform(datum.platform, datum.count);
  };

  const isTickClickable = (index: number) => {
    const datum = data[index];
    if (!datum || !datum.count) return false;
    return (
      getBuiltinPlatformLabelId(builtInLabels, datum.platform) !== undefined
    );
  };

  const containerRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(0);

  useEffect(() => {
    const node = containerRef.current;
    if (!node) return undefined;
    setContainerWidth(node.getBoundingClientRect().width);
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (entry) setContainerWidth(entry.contentRect.width);
    });
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  // Reproduce the sibling checkerboard's cellW clamp on its scroll-wrapper
  // width (our container width minus its y-axis gutter), then scale cellH
  // proportionally and sum up the 8 rows + 7 gaps.
  const wrapperWidth = Math.max(0, containerWidth - CHECKERBOARD_Y_AXIS_WIDTH);
  const idealCellW =
    wrapperWidth > 0
      ? (wrapperWidth - CELL_GAP * (CHECKERBOARD_NUM_COLS - 1)) /
        CHECKERBOARD_NUM_COLS
      : CELL_W;
  const cellW = Math.max(CELL_W, Math.min(idealCellW, CELL_W * CELL_MAX_SCALE));
  const cellH = cellW * (CELL_H / CELL_W);
  const chartHeight =
    cellH * CHECKERBOARD_NUM_ROWS + CELL_GAP * (CHECKERBOARD_NUM_ROWS - 1);

  const tickFontSize = 12;
  const yAxisWidth = containerWidth >= WIDE_YAXIS_THRESHOLD ? 90 : 84;

  return (
    <div className={baseClass} ref={containerRef}>
      <h2 className={`${baseClass}__title`}>Hosts enrolled</h2>
      <div className={`${baseClass}__chart-container`}>
        <ResponsiveContainer width="100%" height={chartHeight}>
          <BarChart
            data={data}
            layout="vertical"
            margin={{ top: 0, right: 20, bottom: 0, left: 0 }}
            barCategoryGap="25%"
          >
            <CartesianGrid
              horizontal={false}
              strokeDasharray="3 3"
              stroke="var(--ui-fleet-black-10)"
            />
            <CartesianGrid
              vertical={false}
              stroke="var(--ui-fleet-black-10)"
              horizontalCoordinatesGenerator={({ offset }) => {
                const { top, height } = offset;
                const bandHeight = height / data.length;
                return data
                  .map((_, i) => top + i * bandHeight)
                  .concat(top + height);
              }}
            />
            <XAxis
              type="number"
              tickFormatter={formatTick}
              axisLine={false}
              tickLine={false}
              tickMargin={6}
              tick={{ fontSize: tickFontSize, fontWeight: 600 }}
              allowDecimals={false}
            />
            <YAxis
              type="category"
              dataKey="label"
              axisLine={false}
              tickLine={false}
              width={yAxisWidth}
              interval={0}
              tick={
                <ClickableYAxisTick
                  fontSize={tickFontSize}
                  isClickable={isTickClickable}
                  onLabelClick={handleTickClick}
                />
              }
            />
            <Tooltip
              content={<HostsEnrolledTooltip />}
              cursor={false}
              isAnimationActive={false}
            />
            <Bar
              dataKey="count"
              radius={[0, 4, 4, 0]}
              barSize={16}
              isAnimationActive={false}
              activeBar={{ fill: BAR_HOVER_COLOR }}
              onClick={(d) => handleBarClick(d.payload as IPlatformDatum)}
            >
              {data.map((entry) => (
                <Cell
                  key={entry.label}
                  fill={BAR_COLOR}
                  className={
                    entry.count > 0 ? `${baseClass}__bar--clickable` : undefined
                  }
                />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};

export default HostsEnrolledCard;
