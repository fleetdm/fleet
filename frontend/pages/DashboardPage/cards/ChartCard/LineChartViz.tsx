import React, { useCallback } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { format, parseISO } from "date-fns";

import { IFormattedDataPoint } from "./types";

const baseClass = "chart-card";

interface ILineChartVizProps {
  data: IFormattedDataPoint[];
  selectedDays: number;
}

// Use the design-system accent token via CSS custom property so recharts
// picks up the themed value for the SVG stroke.
const LINE_STROKE = "var(--core-vibrant-blue)";

const LineChartViz = ({
  data,
  selectedDays,
}: ILineChartVizProps): JSX.Element => {
  const formatXAxis = useCallback(
    (timestamp: string) => {
      try {
        const date = parseISO(timestamp);
        return selectedDays === 1 ? format(date, "ha") : format(date, "MMM d");
      } catch {
        return "";
      }
    },
    [selectedDays]
  );

  const formatYAxisTick = (val: number): string => `${val}%`;

  const renderTooltip = useCallback((props: any) => {
    const { active, payload } = props;
    if (!active || !payload?.length) return null;
    const point = payload[0].payload as IFormattedDataPoint;
    return (
      <div className={`${baseClass}__tooltip`}>
        <div className={`${baseClass}__tooltip-label`}>{point.label}</div>
        <div className={`${baseClass}__tooltip-value`}>
          {point.percentage}% ({point.value.toLocaleString()} hosts)
        </div>
      </div>
    );
  }, []);

  const tickInterval = Math.max(1, Math.floor(data.length / 8));

  return (
    <ResponsiveContainer width="100%" height={280}>
      <LineChart data={data}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} />
        <XAxis
          dataKey="timestamp"
          tickFormatter={formatXAxis}
          interval={tickInterval}
          tick={{ fontSize: 12 }}
        />
        <YAxis
          tick={{ fontSize: 12 }}
          width={50}
          domain={[0, 100]}
          tickFormatter={formatYAxisTick}
        />
        <Tooltip content={renderTooltip} />
        <Line
          type="monotone"
          dataKey="percentage"
          stroke={LINE_STROKE}
          strokeWidth={2}
          dot={false}
          activeDot={{ r: 4 }}
        />
      </LineChart>
    </ResponsiveContainer>
  );
};

export default LineChartViz;
