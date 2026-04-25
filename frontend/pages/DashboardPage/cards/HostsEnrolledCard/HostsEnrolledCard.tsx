import React from "react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
  Cell,
} from "recharts";

const baseClass = "hosts-enrolled-card";

// Use the design-system color token via CSS custom property so recharts
// picks up the themed value for the SVG fill.
const BAR_COLOR = "var(--core-fleet-green)";

export interface IHostPlatformCounts {
  darwin: number;
  windows: number;
  linux: number;
  chrome: number;
  ios: number;
  ipados: number;
  android: number;
}

interface IHostsEnrolledCardProps {
  counts: IHostPlatformCounts;
}

interface IPlatformDatum {
  label: string;
  count: number;
}

const formatTick = (value: number): string => {
  if (value >= 1000) {
    const k = value / 1000;
    return Number.isInteger(k) ? `${k}k` : `${k.toFixed(1)}k`;
  }
  return `${value}`;
};

const HostsEnrolledCard = ({
  counts,
}: IHostsEnrolledCardProps): JSX.Element => {
  const data: IPlatformDatum[] = [
    { label: "macOS", count: counts.darwin },
    { label: "Windows", count: counts.windows },
    { label: "Linux", count: counts.linux },
    { label: "ChromeOS", count: counts.chrome },
    { label: "iOS", count: counts.ios },
    { label: "iPadOS", count: counts.ipados },
    { label: "Android", count: counts.android },
  ];

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
            tick={{ fontSize: 14 }}
          />
          <Bar dataKey="count" radius={[0, 4, 4, 0]} barSize={16}>
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
