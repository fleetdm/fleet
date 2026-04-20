export type ChartType = "line" | "checkerboard";

export interface IDataSet {
  name: string;
  label: string;
  isPercentage: boolean;
  defaultChartType: ChartType;
}

export interface IFormattedDataPoint {
  timestamp: string;
  label: string;
  value: number;
  percentage: number;
}
