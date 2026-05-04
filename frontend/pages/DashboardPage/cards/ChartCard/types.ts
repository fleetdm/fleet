import { ReactNode } from "react";

export type ChartType = "line" | "checkerboard";

export type ChartTheme = "green" | "red";

export interface IDataSet {
  name: string;
  label: string;
  defaultChartType: ChartType;
  description?: ReactNode;
  tooltipFormatter?: TooltipFormatter;
  theme?: ChartTheme;
}

export interface IFormattedDataPoint {
  timestamp: string;
  label: string;
  value: number;
  percentage: number;
  total?: number;
}

export type TooltipFormatter = ({
  value,
  total,
  percentage,
}: {
  value: number;
  total?: number;
  percentage?: number;
}) => string | ReactNode;
