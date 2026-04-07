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
import { SingleValue } from "react-select-5";

import chartsAPI, {
  IChartResponse,
  IChartRequestParams,
  IChartQueryKey,
} from "services/entities/charts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import DataError from "components/DataError";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Icon from "components/Icon";

import ChartFilterModal, { IChartFilterState } from "./ChartFilterModal";

const baseClass = "chart-card";

const DAYS_OPTIONS: CustomOptionType[] = [
  { label: "Last 24 hours", value: "1" },
  { label: "Last 7 days", value: "7" },
  { label: "Last 14 days", value: "14" },
  { label: "Last 30 days", value: "30" },
];

const CHART_TYPE_OPTIONS: CustomOptionType[] = [
  { label: "Check-in activity", value: "uptime" },
  { label: "Policy compliance", value: "policy", isDisabled: true },
  { label: "Vulnerabilities", value: "cve", isDisabled: true },
];

interface IFormattedDataPoint {
  timestamp: string;
  label: string;
  value: number;
  percentage: number;
}

const ChartCard = (): JSX.Element => {
  const [selectedDays, setSelectedDays] = useState(7);
  const [selectedMetric, setSelectedMetric] = useState("uptime");
  const [filterParams, setFilterParams] = useState<IChartRequestParams>({});
  const [showFilterModal, setShowFilterModal] = useState(false);
  const [chartFilters, setChartFilters] = useState<IChartFilterState>({
    labelIDs: [],
    platforms: [],
  });

  const queryParams: IChartRequestParams = useMemo(
    () => ({
      ...filterParams,
      days: selectedDays,
      label_ids: chartFilters.labelIDs.length
        ? chartFilters.labelIDs.join(",")
        : undefined,
      platforms: chartFilters.platforms.length
        ? chartFilters.platforms.join(",")
        : undefined,
    }),
    [filterParams, selectedDays, chartFilters]
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
      const labelFormat = selectedDays === 1 ? "h:mm a" : "MMM d, h:mm a";
      return {
        timestamp: point.timestamp,
        label: format(date, labelFormat),
        value: point.value,
        percentage: Math.round((point.value / totalHosts) * 100),
      };
    });
  }, [chartData, selectedDays]);

  const renderTooltip = useCallback((props: any) => {
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
  }, []);

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

    const tickInterval = Math.max(1, Math.floor(formattedData.length / 8));

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

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__header-left`}>
          <DropdownWrapper
            name="days-range"
            value={String(selectedDays)}
            options={DAYS_OPTIONS}
            onChange={(option: SingleValue<CustomOptionType>) => {
              if (option) {
                setSelectedDays(Number(option.value));
              }
            }}
            className={`${baseClass}__days-dropdown`}
          />
        </div>
        <div className={`${baseClass}__header-right`}>
          <DropdownWrapper
            name="chart-type"
            value={selectedMetric}
            options={CHART_TYPE_OPTIONS}
            onChange={(option: SingleValue<CustomOptionType>) => {
              if (option) {
                setSelectedMetric(option.value);
              }
            }}
            className={`${baseClass}__chart-type-dropdown`}
          />
          <button
            type="button"
            className={`${baseClass}__settings-btn`}
            onClick={() => setShowFilterModal(true)}
          >
            <Icon name="settings" />
          </button>
        </div>
      </div>
      <div className={`${baseClass}__chart-container`}>{renderChart()}</div>
      {showFilterModal && (
        <ChartFilterModal
          filters={chartFilters}
          onApply={(newFilters) => {
            setChartFilters(newFilters);
            setShowFilterModal(false);
          }}
          onCancel={() => setShowFilterModal(false)}
        />
      )}
    </div>
  );
};

export default ChartCard;
