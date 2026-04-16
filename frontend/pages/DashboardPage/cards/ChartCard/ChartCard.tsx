import React, { useState, useMemo } from "react";
import { useQuery } from "react-query";
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
import LineChartViz from "./LineChartViz";
import CheckerboardViz from "./CheckerboardViz";
import ComplianceStackedBarViz from "./ComplianceStackedBarViz";
import { IFormattedDataPoint } from "./types";

const baseClass = "chart-card";

const DATASETS: CustomOptionType[] = [
  { value: "uptime", label: "Check-in activity" },
  { value: "policy_failing", label: "Hosts failing policies" },
  { value: "cve", label: "Vulnerabilities" },
];

const hasActiveFilters = (filters: IChartFilterState): boolean => {
  return (
    filters.labelIDs.length > 0 ||
    filters.platforms.length > 0 ||
    filters.selectedHosts.length > 0
  );
};

const ChartCard = (): JSX.Element => {
  const [selectedDays] = useState(30);
  const [selectedMetric, setSelectedMetric] = useState("uptime");
  const [showFilterModal, setShowFilterModal] = useState(false);
  const [chartFilters, setChartFilters] = useState<IChartFilterState>({
    labelIDs: [],
    platforms: [],
    hostFilterMode: "none",
    selectedHosts: [],
  });

  const queryParams: IChartRequestParams = useMemo(() => {
    let downsample: number | undefined;
    if (selectedDays === 30) {
      downsample = 3;
    } else if (selectedDays >= 7) {
      downsample = 2;
    }
    return {
      days: selectedDays,
      downsample,
      tz_offset: new Date().getTimezoneOffset(),
      label_ids: chartFilters.labelIDs.length
        ? chartFilters.labelIDs.join(",")
        : undefined,
      platforms: chartFilters.platforms.length
        ? chartFilters.platforms.join(",")
        : undefined,
      include_host_ids:
        chartFilters.hostFilterMode === "include" &&
          chartFilters.selectedHosts.length
          ? chartFilters.selectedHosts.map((h) => h.id).join(",")
          : undefined,
      exclude_host_ids:
        chartFilters.hostFilterMode === "exclude" &&
          chartFilters.selectedHosts.length
          ? chartFilters.selectedHosts.map((h) => h.id).join(",")
          : undefined,
    };
  }, [selectedDays, chartFilters]);

  const {
    data: chartData,
    isFetching,
    error,
  } = useQuery<IChartResponse, Error, IChartResponse, IChartQueryKey[]>(
    [{ scope: "chart", metric: selectedMetric, params: queryParams }],
    () => chartsAPI.getChartData(selectedMetric, queryParams),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      staleTime: 300000, // 5 minutes
    }
  );

  const formattedData: IFormattedDataPoint[] = useMemo(() => {
    if (!chartData?.data || !chartData?.series) return [];
    const totalHosts = chartData.total_hosts || 1;
    const primaryKey = chartData.series[0]?.key ?? "total";
    return chartData.data.map((point) => {
      const date = parseISO(point.timestamp);
      const labelFormat = selectedDays === 1 ? "h:mm a" : "MMM d, h:mm a";
      const value = point.values[primaryKey] ?? 0;
      return {
        timestamp: point.timestamp,
        label: format(date, labelFormat),
        value,
        percentage: Math.round((value / totalHosts) * 100),
      };
    });
  }, [chartData, selectedDays]);

  const renderChart = () => {
    if (isFetching) {
      return <Spinner includeContainer={false} verticalPadding="small" />;
    }
    if (error) {
      return <DataError />;
    }
    if (!chartData?.data?.length) {
      return (
        <div className={`${baseClass}__no-data`}>
          No chart data available yet.
        </div>
      );
    }

    switch (chartData.visualization) {
      case "stacked_bar":
        return (
          <ComplianceStackedBarViz
            series={chartData.series}
            data={chartData.data}
          />
        );
      case "checkerboard":
        return (
          <CheckerboardViz
            data={formattedData}
            selectedDays={selectedDays}
            isPercentage
          />
        );
      case "line":
      default:
        return (
          <LineChartViz
            data={formattedData}
            selectedDays={selectedDays}
            isPercentage={false}
          />
        );
    }
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__header-left`}>
          <DropdownWrapper
            name="dataset"
            value={selectedMetric}
            options={DATASETS}
            onChange={(option: SingleValue<CustomOptionType>) => {
              if (option) {
                setSelectedMetric(option.value);
              }
            }}
            className={`${baseClass}__dataset-dropdown`}
          />
          {hasActiveFilters(chartFilters) && (
            <span className={`${baseClass}__filtered-badge`}>Filtered</span>
          )}
        </div>
        <div className={`${baseClass}__header-right`}>
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
