import React, { useMemo, useState } from "react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
} from "recharts";
import { format, parseISO } from "date-fns";

import { IChartDataPoint, ISeriesMeta } from "services/entities/charts";

const baseClass = "compliance-stacked-bar-viz";

const TEAM_COLORS = [
  "#6F5BFF", // violet
  "#00A382", // teal
  "#F7A12A", // orange
  "#3DB5E6", // light blue
  "#C23838", // red
  "#8B5FBF", // purple
  "#2B7A78", // dark teal
  "#E6A700", // amber
  "#505A75", // slate
  "#A05252", // rust
];

const EMPTY_TEAM_COLOR = "#D5D8E0";

interface IComplianceStackedBarVizProps {
  series: ISeriesMeta[];
  data: IChartDataPoint[];
}

const formatPercent = (v: number): string =>
  `${Math.round(v * 1000) / 10}%`;

const ComplianceStackedBarViz = ({
  series,
  data,
}: IComplianceStackedBarVizProps): JSX.Element => {
  const [hoveredSeries, setHoveredSeries] = useState<string | null>(null);

  // Recharts wants flat rows with each series key as a top-level property.
  const chartData = useMemo(() => {
    return data.map((point) => {
      const row: Record<string, string | number> = {
        timestamp: point.timestamp,
      };
      Object.entries(point.values).forEach(([key, count]) => {
        row[key] = count;
      });
      return row;
    });
  }, [data]);

  const formatXAxis = (ts: string): string => {
    try {
      return format(parseISO(ts), "MMM d");
    } catch {
      return ts;
    }
  };

  if (!series.length || !chartData.length) {
    return (
      <div className={`${baseClass}__no-data`}>
        No compliance data yet.
      </div>
    );
  }

  return (
    <div className={`${baseClass}__body-grid`}>
      <div className={`${baseClass}__chart`}>
        <ResponsiveContainer width="100%" height={320}>
          <BarChart
            data={chartData}
            margin={{ top: 8, right: 8, left: 0, bottom: 8 }}
          >
            <CartesianGrid strokeDasharray="2 3" vertical={false} />
            <XAxis
              dataKey="timestamp"
              tickFormatter={formatXAxis}
              tick={{ fontSize: 11, fill: "#8B8FA2" }}
              axisLine={{ stroke: "#515774" }}
              tickLine={false}
              interval="preserveStartEnd"
              minTickGap={40}
            />
            <YAxis
              tick={{ fontSize: 11, fill: "#8B8FA2" }}
              axisLine={{ stroke: "#515774" }}
              tickLine={false}
              allowDecimals={false}
            />
            {series.map((s, idx) => {
              const hostCount = (s.stats?.host_count as number) ?? 0;
              const color =
                hostCount === 0
                  ? EMPTY_TEAM_COLOR
                  : TEAM_COLORS[idx % TEAM_COLORS.length];
              const isDimmed =
                hoveredSeries !== null && hoveredSeries !== s.key;
              return (
                <Bar
                  key={s.key}
                  dataKey={s.key}
                  stackId="a"
                  fill={color}
                  fillOpacity={isDimmed ? 0.25 : 1}
                  isAnimationActive={false}
                />
              );
            })}
          </BarChart>
        </ResponsiveContainer>
      </div>
      <ul className={`${baseClass}__legend`}>
        <li className={`${baseClass}__legend-header`}>
          FLEETS (compliance %)
        </li>
        {series.map((s, idx) => {
          const hostCount = (s.stats?.host_count as number) ?? 0;
          const isEmpty = hostCount === 0;
          const color = isEmpty
            ? EMPTY_TEAM_COLOR
            : TEAM_COLORS[idx % TEAM_COLORS.length];
          const pct = s.stats?.fully_compliant_pct as number | undefined;
          const hostsFailing = (s.stats?.hosts_failing_any as number) ?? 0;
          const policiesFailing = (s.stats?.policies_failing as number) ?? 0;
          const policiesTracked = (s.stats?.policies_tracked as number) ?? 0;
          return (
            <li
              key={s.key}
              className={`${baseClass}__legend-item ${
                isEmpty ? `${baseClass}__legend-item--empty` : ""
              }`}
              onMouseEnter={() => setHoveredSeries(s.key)}
              onMouseLeave={() => setHoveredSeries(null)}
            >
              <span
                className={`${baseClass}__swatch`}
                style={{ backgroundColor: color }}
              />
              <span className={`${baseClass}__team-name`}>{s.label}</span>
              <span className={`${baseClass}__team-pct`}>
                {isEmpty || pct === undefined
                  ? "—"
                  : `(${formatPercent(pct)} compliant)`}
              </span>
              {!isEmpty && hoveredSeries === s.key && (
                <div className={`${baseClass}__legend-tooltip`}>
                  {hostsFailing} of {hostCount} hosts failing
                  <br />
                  {policiesFailing}/{policiesTracked} policies failing
                </div>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
};

export default ComplianceStackedBarViz;
