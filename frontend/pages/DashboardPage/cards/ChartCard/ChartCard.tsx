import React, { useState, useCallback, useMemo } from "react";
import { useQuery } from "react-query";
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

import chartsAPI, {
  IChartResponse,
  IChartRequestParams,
  IChartQueryKey,
} from "services/entities/charts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import DataError from "components/DataError";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "chart-card";

const DAYS_OPTIONS = [
  { label: "1d", value: 1 },
  { label: "7d", value: 7 },
  { label: "14d", value: 14 },
  { label: "30d", value: 30 },
];

const DATASET_OPTIONS = [{ label: "Check-in activity", value: "uptime" }];

interface IChartCardProps {
  onOpenFilters?: () => void;
}

interface IFormattedDataPoint {
  timestamp: string;
  label: string;
  value: number;
  percentage: number;
}

const ChartCard = ({ onOpenFilters }: IChartCardProps): JSX.Element => {
  const [selectedDays, setSelectedDays] = useState(7);
  const [selectedMetric, setSelectedMetric] = useState("uptime");
  const [filterParams, setFilterParams] = useState<IChartRequestParams>({});

  const queryParams: IChartRequestParams = useMemo(
    () => ({
      ...filterParams,
      days: selectedDays,
    }),
    [filterParams, selectedDays]
  );

  const { data: chartData, isFetching, error } = useQuery<
    IChartResponse,
    Error,
    IChartResponse,
    IChartQueryKey[]
  >(
    [{ scope: "chart", metric: selectedMetric, params: queryParams }],
    () => chartsAPI.getChartData(selectedMetric, queryParams),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      staleTime: 300000, // 5 minutes
    }
  );

  const formattedData: IFormattedDataPoint[] = useMemo(() => {
    if (!chartData?.data) return [];
    const totalHosts = chartData.total_hosts || 1;
    return chartData.data.map((point) => {
      const date = parseISO(point.timestamp);
      const labelFormat =
        selectedDays === 1 ? "h:mm a" : "MMM d, h:mm a";
      return {
        timestamp: point.timestamp,
        label: format(date, labelFormat),
        value: point.value,
        percentage: Math.round((point.value / totalHosts) * 100),
      };
    });
  }, [chartData, selectedDays]);

  const handleDaysChange = useCallback((days: number) => {
    setSelectedDays(days);
  }, []);

  const renderTooltip = useCallback(
    (props: any) => {
      const { active, payload } = props;
      if (!active || !payload?.length) return null;
      const data = payload[0].payload as IFormattedDataPoint;
      return (
        <div className={`${baseClass}__tooltip`}>
          <div className={`${baseClass}__tooltip-label`}>{data.label}</div>
          <div className={`${baseClass}__tooltip-value`}>
            {data.value.toLocaleString()} hosts ({data.percentage}%)
          </div>
        </div>
      );
    },
    []
  );

  const formatXAxis = useCallback(
    (timestamp: string) => {
      try {
        const date = parseISO(timestamp);
        return selectedDays === 1
          ? format(date, "ha")
          : format(date, "MMM d");
      } catch {
        return "";
      }
    },
    [selectedDays]
  );

  const renderChart = () => {
    if (isFetching) {
      return <Spinner includeContainer={false} verticalPadding="small" />;
    }
    if (error) {
      return <DataError />;
    }
    if (!formattedData.length) {
      return (
        <div className={`${baseClass}__no-data`}>
          No chart data available yet.
        </div>
      );
    }

    // Calculate tick interval to avoid overcrowding.
    const tickInterval = Math.max(
      1,
      Math.floor(formattedData.length / 8)
    );

    return (
      <ResponsiveContainer width="100%" height={280}>
        <LineChart data={formattedData}>
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
            tickFormatter={(val: number) =>
              val >= 1000 ? `${(val / 1000).toFixed(1)}k` : String(val)
            }
          />
          <Tooltip content={renderTooltip} />
          <Line
            type="monotone"
            dataKey="value"
            stroke="#6A67CE"
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 4 }}
          />
        </LineChart>
      </ResponsiveContainer>
    );
  };

  const selectedDatasetLabel =
    DATASET_OPTIONS.find((d) => d.value === selectedMetric)?.label ??
    selectedMetric;

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__header-left`}>
          <h2 className={`${baseClass}__title`}>{selectedDatasetLabel}</h2>
          {chartData && (
            <span className={`${baseClass}__total-hosts`}>
              {chartData.total_hosts.toLocaleString()} total hosts
            </span>
          )}
        </div>
        <div className={`${baseClass}__header-right`}>
          {onOpenFilters && (
            <Button
              variant="text-icon"
              onClick={onOpenFilters}
              className={`${baseClass}__filter-btn`}
            >
              <Icon name="filter" />
            </Button>
          )}
          <div className={`${baseClass}__days-selector`}>
            {DAYS_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                type="button"
                className={`${baseClass}__days-btn ${
                  selectedDays === opt.value
                    ? `${baseClass}__days-btn--active`
                    : ""
                }`}
                onClick={() => handleDaysChange(opt.value)}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>
      </div>
      <div className={`${baseClass}__chart-container`}>{renderChart()}</div>
    </div>
  );
};

export default ChartCard;
