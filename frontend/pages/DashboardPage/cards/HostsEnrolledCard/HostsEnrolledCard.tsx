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

import PATHS from "router/paths";
import { ILabelSummary } from "interfaces/label";
import { getPathWithQueryParams } from "utilities/url";
import { PLATFORM_NAME_TO_LABEL_NAME } from "pages/DashboardPage/helpers";

const baseClass = "hosts-enrolled-card";

// Use design-system color tokens via CSS custom properties so recharts picks
// up the themed values for the SVG fills.
const BAR_COLOR = "var(--core-fleet-green)";
const BAR_HOVER_COLOR = "var(--core-fleet-green-over)";

// Match the checkerboard's cell-grid height so the bar plot lines up with the
// cells next to it (excluding the checkerboard's x-axis labels and legend).
// Values come from CheckerboardViz: cellH * numRows + CELL_GAP * (numRows - 1)
// for the 30-day view (numRows = 8).
const CHART_HEIGHT_NARROW = 190; // 19 * 8 + 2 * 7
const CHART_HEIGHT_WIDE = 242; // 28.5 * 8 + 2 * 7
const WIDE_THRESHOLD = 700; // mirror CheckerboardViz

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
  builtInLabels?: ILabelSummary[];
  currentTeamId?: number;
  router: InjectedRouter;
}

interface IPlatformDatum {
  label: string;
  count: number;
  platform: PlatformKey;
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
  return (
    <div className={`${baseClass}__tooltip`}>
      <div className={`${baseClass}__tooltip-label`}>{datum.label}</div>
      <div className={`${baseClass}__tooltip-value`}>
        {datum.count.toLocaleString()} hosts
      </div>
    </div>
  );
};

interface IYAxisTickProps {
  x?: number;
  y?: number;
  payload?: { value: string; index: number };
  fontSize: number;
  isClickable: (index: number) => boolean;
  onLabelClick: (index: number) => void;
}

const ClickableYAxisTick = ({
  x = 0,
  y = 0,
  payload,
  fontSize,
  isClickable,
  onLabelClick,
}: IYAxisTickProps): JSX.Element => {
  if (!payload) return <g />;
  const clickable = isClickable(payload.index);
  return (
    <g
      transform={`translate(${x},${y})`}
      onClick={clickable ? () => onLabelClick(payload.index) : undefined}
    >
      <text
        x={0}
        y={0}
        dy={4}
        textAnchor="end"
        fontSize={fontSize}
        className={clickable ? `${baseClass}__tick--clickable` : undefined}
      >
        {payload.value}
      </text>
    </g>
  );
};

const HostsEnrolledCard = ({
  counts,
  builtInLabels,
  currentTeamId,
  router,
}: IHostsEnrolledCardProps): JSX.Element => {
  const data: IPlatformDatum[] = PLATFORM_ROWS.map(({ platform, label }) => ({
    platform,
    label,
    count: counts[platform],
  }));

  // Given a platform, find the corresponding built-in label ID for linking to the
  // hosts list.
  const getLabelId = (platform: PlatformKey): number | undefined => {
    const labelName = PLATFORM_NAME_TO_LABEL_NAME[platform];
    return builtInLabels?.find((l) => l.name === labelName)?.id;
  };

  const navigateToPlatform = (platform: PlatformKey, count: number) => {
    if (!count) return;
    const labelId = getLabelId(platform);
    if (labelId === undefined) return;
    router.push(
      getPathWithQueryParams(PATHS.MANAGE_HOSTS_LABEL(labelId), {
        fleet_id: currentTeamId,
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
    return getLabelId(datum.platform) !== undefined;
  };

  // Mirror CheckerboardViz's wide-mode detection so the bar chart's plot area
  // matches the cell grid height in both layouts.
  const containerRef = useRef<HTMLDivElement>(null);
  const [isWide, setIsWide] = useState(false);

  useEffect(() => {
    const node = containerRef.current;
    if (!node) return undefined;
    setIsWide(node.getBoundingClientRect().width >= WIDE_THRESHOLD);
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (entry) setIsWide(entry.contentRect.width >= WIDE_THRESHOLD);
    });
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  const chartHeight = isWide ? CHART_HEIGHT_WIDE : CHART_HEIGHT_NARROW;
  // 7 platforms in ~166px (narrow) leaves ~23px per row, so the default 14px
  // ticks crowd. Step down a couple sizes when narrow.
  const tickFontSize = isWide ? 14 : 11;
  // ChromeOS is the widest label and just barely doesn't fit at 80/60, so add
  // a bit of breathing room.
  const yAxisWidth = isWide ? 90 : 68;

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
            <CartesianGrid horizontal={false} strokeDasharray="3 3" />
            <CartesianGrid
              vertical={false}
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
              tick={{ fontSize: tickFontSize }}
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
