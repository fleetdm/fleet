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
import { IDataSet, IFormattedDataPoint } from "./types";

const baseClass = "chart-card";

const DAYS_OPTIONS: CustomOptionType[] = [
  { label: "Last 24 hours", value: "1" },
  { label: "Last 7 days", value: "7" },
  { label: "Last 14 days", value: "14" },
  { label: "Last 30 days", value: "30" },
];

const DATASETS: IDataSet[] = [
  {
    name: "uptime",
    label: "Check-in activity",
    isPercentage: true,
    defaultChartType: "checkerboard",
  },
  {
    name: "policy",
    label: "Policy compliance",
    isPercentage: true,
    defaultChartType: "line",
  },
  {
    name: "cve",
    label: "Vulnerabilities",
    isPercentage: false,
    defaultChartType: "line",
  },
];

const DATASET_OPTIONS: CustomOptionType[] = DATASETS.map((ds) => ({
  label: ds.label,
  value: ds.name,
}));

const getDataset = (name: string): IDataSet =>
  DATASETS.find((ds) => ds.name === name) || DATASETS[0];

const ChartCard = (): JSX.Element => {
  const [selectedDays, setSelectedDays] = useState(30);
  const [selectedMetric, setSelectedMetric] = useState("uptime");
  const [filterParams, setFilterParams] = useState<IChartRequestParams>({});
  const [showFilterModal, setShowFilterModal] = useState(false);
  const [chartFilters, setChartFilters] = useState<IChartFilterState>({
    labelIDs: [],
    platforms: [],
    hostFilterMode: "none",
    selectedHosts: [],
  });

  const currentDataset = getDataset(selectedMetric);

  const queryParams: IChartRequestParams = useMemo(() => {
    let downsample: number | undefined;
    if (selectedDays === 30) {
      downsample = 3;
    } else if (selectedDays >= 7) {
      downsample = 2;
    }
    return {
      ...filterParams,
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
  }, [filterParams, selectedDays, chartFilters]);

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

    const vizProps = {
      data: formattedData,
      selectedDays,
      isPercentage: currentDataset.isPercentage,
    };

    switch (currentDataset.defaultChartType) {
      case "checkerboard":
        return <CheckerboardViz {...vizProps} />;
      case "line":
      default:
        return <LineChartViz {...vizProps} />;
    }
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
            name="dataset"
            value={selectedMetric}
            options={DATASET_OPTIONS}
            onChange={(option: SingleValue<CustomOptionType>) => {
              if (option) {
                setSelectedMetric(option.value);
              }
            }}
            className={`${baseClass}__dataset-dropdown`}
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
