import React from "react";
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
  isClickable: (index: number) => boolean;
  onLabelClick: (index: number) => void;
}

const ClickableYAxisTick = ({
  x = 0,
  y = 0,
  payload,
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
        fontSize={14}
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

  return (
    <div className={baseClass}>
      <h2 className={`${baseClass}__title`}>Hosts enrolled</h2>
      <ResponsiveContainer width="100%" height={280}>
        <BarChart
          data={data}
          layout="vertical"
          margin={{ top: 0, right: 20, bottom: 0, left: 0 }}
          barCategoryGap="25%"
        >
          <CartesianGrid horizontal={false} strokeDasharray="3 3" />
          <CartesianGrid vertical={false} />
          <XAxis
            type="number"
            tickFormatter={formatTick}
            axisLine={false}
            tickLine={false}
            tick={{ fontSize: 14 }}
          />
          <YAxis
            type="category"
            dataKey="label"
            axisLine={false}
            tickLine={false}
            width={80}
            tick={
              <ClickableYAxisTick
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
  );
};

export default HostsEnrolledCard;
