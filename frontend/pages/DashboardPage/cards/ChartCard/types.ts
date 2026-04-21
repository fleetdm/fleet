export type ChartType = "line" | "checkerboard";

export interface IDataSet {
  name: string;
  label: string;
  defaultChartType: ChartType;
  description?: string;
}

export interface IFormattedDataPoint {
  timestamp: string;
  label: string;
  value: number;
  percentage: number;
}
