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
  relativeScale?: boolean;
}

// CVE chart software categories. Values must match the backend api.CVECategory*
// keys. The "os" category covers operating-system vulnerabilities and the Linux
// kernel. Descriptions mirror the Figma sub-labels.
export const CVE_SOFTWARE_CATEGORIES = [
  {
    value: "os",
    label: "Operating system (OS)",
    tooltipLabel: "OS",
    description: "",
  },
  {
    value: "browsers",
    label: "Browsers",
    tooltipLabel: "browsers",
    description: "Google Chrome, Safari, Mozilla Firefox, Brave, and Opera",
  },
  {
    value: "office",
    label: "Microsoft Office",
    tooltipLabel: "Microsoft Office",
    description: "Word, Excel, PowerPoint, and Outlook",
  },
  {
    value: "adobe",
    label: "Adobe",
    tooltipLabel: "Adobe",
    description: "Acrobat, Flash, and Shockwave Player",
  },
] as const;

export const ALL_CVE_SOFTWARE_CATEGORY_VALUES = CVE_SOFTWARE_CATEGORIES.map(
  (c) => c.value as string
);

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

export const HISTORICAL_DATA_CONFIG_KEYS = [
  "uptime",
  "vulnerabilities",
] as const;

export type HistoricalDataConfigKey = typeof HISTORICAL_DATA_CONFIG_KEYS[number];

export interface IHistoricalDataSettings {
  uptime: boolean;
  vulnerabilities: boolean;
}

// Maps internal dataset names (used by the chart API) to the config keys
// surfaced in features.historical_data. Datasets not present here are
// treated as having no toggle (collection implicitly enabled).
export const DATASET_CONFIG_KEY: Partial<
  Record<string, HistoricalDataConfigKey>
> = {
  uptime: "uptime",
  cve: "vulnerabilities",
};

export const DATASET_LABEL: Record<HistoricalDataConfigKey, string> = {
  uptime: "Hosts online",
  vulnerabilities: "Vulnerability exposure",
};

// Applies the global-AND-fleet precedence rule for historical data
// collection. Missing settings (e.g. no fleet selected) are treated as
// enabled so the rule is a true AND with no surprises.
export const isHistoricalDataEnabled = (
  global: IHistoricalDataSettings | undefined,
  fleet: IHistoricalDataSettings | undefined,
  configKey: HistoricalDataConfigKey
): boolean => {
  const g = global?.[configKey] ?? true;
  const f = fleet?.[configKey] ?? true;
  return g && f;
};
