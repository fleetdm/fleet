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
} from "recharts";

import PATHS from "router/paths";
import { ILabelSummary } from "interfaces/label";
import { getPathWithQueryParams } from "utilities/url";
import { PLATFORM_NAME_TO_LABEL_NAME } from "pages/DashboardPage/helpers";

const baseClass = "hosts-enrolled-card";

// Use design-system color tokens via CSS custom properties so recharts picks
// up the themed value for SVG fill / text colors.
const BAR_COLOR = "var(--core-fleet-green)";
const TICK_COLOR = "var(--ui-fleet-black-75)";

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

interface IYAxisTickProps {
  x?: number;
  y?: number;
  payload?: { value: string; index: number };
  onLabelClick: (index: number) => void;
}

const ClickableYAxisTick = ({
  x = 0,
  y = 0,
  payload,
  onLabelClick,
}: IYAxisTickProps): JSX.Element => {
  if (!payload) return <g />;
  return (
    <g
      transform={`translate(${x},${y})`}
      style={{ cursor: "pointer", outline: "none" }}
      onClick={() => onLabelClick(payload.index)}
    >
      <text x={0} y={0} dy={4} textAnchor="end" fill={TICK_COLOR} fontSize={14}>
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
            tick={{ fontSize: 14, fill: TICK_COLOR }}
          />
          <YAxis
            type="category"
            dataKey="label"
            axisLine={false}
            tickLine={false}
            width={80}
            tick={<ClickableYAxisTick onLabelClick={handleTickClick} />}
          />
          <Bar
            dataKey="count"
            radius={[0, 4, 4, 0]}
            barSize={16}
            onClick={(d) => handleBarClick(d as IPlatformDatum)}
            style={{ cursor: "pointer", outline: "none" }}
          >
            {data.map((entry) => (
              <Cell key={entry.label} fill={BAR_COLOR} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
};

export default HostsEnrolledCard;
