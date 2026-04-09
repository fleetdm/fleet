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

const BAR_COLOR = "#009a7d";

interface IHostsEnrolledCardProps {
  macCount: number;
  windowsCount: number;
  linuxCount: number;
  chromeCount: number;
  iosCount: number;
  ipadosCount: number;
  androidCount: number;
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
  macCount,
  windowsCount,
  linuxCount,
  chromeCount,
  iosCount,
  ipadosCount,
  androidCount,
}: IHostsEnrolledCardProps): JSX.Element => {
  const data: IPlatformDatum[] = [
    { label: "macOS", count: macCount },
    { label: "Windows", count: windowsCount },
    { label: "Linux", count: linuxCount },
    { label: "ChromeOS", count: chromeCount },
    { label: "iOS", count: iosCount },
    { label: "iPadOS", count: ipadosCount },
    { label: "Android", count: androidCount },
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
