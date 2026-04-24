import { ReactNode } from "react";

export type ChartType = "line" | "checkerboard";

export interface IDataSet {
  name: string;
  label: string;
  defaultChartType: ChartType;
  description?: ReactNode;
}

export interface IFormattedDataPoint {
  timestamp: string;
  label: string;
  value: number;
  percentage: number;
}
