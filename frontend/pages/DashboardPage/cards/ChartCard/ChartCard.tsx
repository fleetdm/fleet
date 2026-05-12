import React, { useEffect, useState, useMemo } from "react";
import { useQuery } from "react-query";
import { format, parseISO } from "date-fns";
import { SingleValue } from "react-select-5";

import chartsAPI, {
  IChartResponse,
  IChartRequestParams,
  IChartQueryKey,
} from "services/entities/charts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";

import {
  IDataSet,
  IFormattedDataPoint,
  DATASET_CONFIG_KEY,
  DATASET_LABEL,
  HistoricalDataConfigKey,
} from "interfaces/charts";

import ChartFilterModal, { IChartFilterState } from "./ChartFilterModal";
import LineChartViz from "./LineChartViz";
import CheckerboardViz from "./CheckerboardViz";
import DataCollectionDisabledState from "./DataCollectionDisabledState";

const baseClass = "chart-card";

// All charts are currently fixed at a 30-day window. When the server supports
// configurable ranges we'll add UI and request-param plumbing for this.
const CHART_DAYS = 30;

const DATASETS: IDataSet[] = [
  {
    name: "uptime",
    label: "Hosts online",
    defaultChartType: "checkerboard",
    description: (
      <>
        The number of hosts detected online (checking in to Fleet) during 
        a given hour.
        <br />
        <br />
        Currently, only macOS, Windows, Linux, and ChromeOS are supported.
      </>
    ),
    tooltipFormatter: ({ value }: { value: number }) =>
      `${value.toLocaleString()} host${value === 1 ? "" : "s"} online`,
    relativeScale: true,
  },
  {
    name: "cve",
    label: "Vulnerability exposure",
    defaultChartType: "checkerboard",
    description: (
      <>
        The number of hosts with critical vulnerabilities detected in browsers
        and{" "}
        <CustomLink
          newTab
          text="other common software "
          variant="tooltip-link"
          url="https://fleetdm.com/learn-more-about/vulnerability-exposure-cves"
        />
        <br />
        <br />
        Want more control? Comprehensive vulnerability filtering is{" "}
        <CustomLink
          newTab
          text="coming soon "
          variant="tooltip-link"
          url="https://github.com/fleetdm/fleet/issues/44746"
        />
      </>
    ),
    tooltipFormatter: ({ value }: { value: number }) =>
      `${value.toLocaleString()} host${value === 1 ? "" : "s"}`,
    theme: "red",
    relativeScale: true,
  },
];

const DATASET_OPTIONS: CustomOptionType[] = DATASETS.map((ds) => ({
  label: ds.label,
  value: ds.name,
}));

const getDataset = (name: string): IDataSet =>
  DATASETS.find((ds) => ds.name === name) || DATASETS[0];

const hasActiveFilters = (filters: IChartFilterState): boolean => {
  const hasHostFilter =
    filters.hostFilterMode !== "none" && filters.selectedHosts.length > 0;
  return (
    filters.labelIDs.length > 0 || filters.platforms.length > 0 || hasHostFilter
  );
};

interface IChartCardProps {
  currentTeamId?: number;
  historicalDataEnabled?: Record<HistoricalDataConfigKey, boolean>;
}

const ChartCard = ({
  currentTeamId,
  historicalDataEnabled,
}: IChartCardProps): JSX.Element => {
  const [selectedMetric, setSelectedMetric] = useState("uptime");
  const [showFilterModal, setShowFilterModal] = useState(false);
  const [chartFilters, setChartFilters] = useState<IChartFilterState>({
    labelIDs: [],
    platforms: [],
    hostFilterMode: "none",
    selectedHosts: [],
  });

  // Labels and selected hosts are team-scoped, so clear filters when the
  // active fleet changes to avoid submitting stale IDs under the new scope.
  useEffect(() => {
    setChartFilters({
      labelIDs: [],
      platforms: [],
      hostFilterMode: "none",
      selectedHosts: [],
    });
  }, [currentTeamId]);

  const currentDataset = getDataset(selectedMetric);

  const datasetConfigKey = DATASET_CONFIG_KEY[currentDataset.name];
  // If a dataset has no config-key mapping (future addition), treat it as
  // enabled — collection toggles only apply to known config keys.
  const datasetCollectionEnabled =
    datasetConfigKey === undefined
      ? true
      : historicalDataEnabled?.[datasetConfigKey] ?? true;

  const queryParams: IChartRequestParams = useMemo(() => {
    return {
      // Add an extra day to ensure we get the full # of calendar days
      // represented in the chart, regardless of timezone.
      days: CHART_DAYS + 1,
      tz_offset: new Date().getTimezoneOffset(),
      fleet_id: currentTeamId,
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
  }, [chartFilters, currentTeamId]);

  const { data: chartData, isLoading, error } = useQuery<
    IChartResponse,
    Error,
    IChartResponse,
    IChartQueryKey[]
  >(
    [{ scope: "chart", metric: selectedMetric, params: queryParams }],
    () => chartsAPI.getChartData(selectedMetric, queryParams),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: datasetCollectionEnabled,
      staleTime: 300000, // 5 minutes
    }
  );

  const formattedData: IFormattedDataPoint[] = useMemo(() => {
    if (!chartData?.data) return [];
    const totalHosts = chartData.total_hosts;
    return chartData.data.map((point) => {
      const date = parseISO(point.timestamp);
      return {
        timestamp: point.timestamp,
        label: format(date, "MMM d, h:mm a"),
        value: point.value,
        percentage: totalHosts
          ? Math.round((point.value / totalHosts) * 100)
          : 0,
        total: totalHosts,
      };
    });
  }, [chartData]);

  const renderChart = () => {
    if (!datasetCollectionEnabled && datasetConfigKey !== undefined) {
      return (
        <DataCollectionDisabledState
          datasetLabel={DATASET_LABEL[datasetConfigKey]}
          currentTeamId={currentTeamId}
        />
      );
    }
    if (isLoading) {
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
      selectedDays: CHART_DAYS,
      theme: currentDataset.theme,
      tooltipFormatter: currentDataset.tooltipFormatter,
      relativeScale: currentDataset.relativeScale,
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
          {DATASET_OPTIONS.length > 1 ? (
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
              nowrapMenu
            />
          ) : (
            <h2 className={`${baseClass}__title`}>{currentDataset.label}</h2>
          )}
          {currentDataset.description && (
            <TooltipWrapper
              tipContent={currentDataset.description}
              position="top"
              underline={false}
              showArrow
              tipOffset={8}
              className={`${baseClass}__description-tooltip`}
            >
              <Icon name="info-outline" />
            </TooltipWrapper>
          )}
          {hasActiveFilters(chartFilters) && (
            <span className={`${baseClass}__filtered-badge`}>Filtered</span>
          )}
        </div>
        <div className={`${baseClass}__header-right`}>
          <Button
            type="button"
            variant="inverse"
            className={`${baseClass}__settings-btn`}
            ariaLabel="Configure chart filters"
            onClick={() => setShowFilterModal(true)}
          >
            <Icon name="settings" />
          </Button>
        </div>
      </div>
      <div className={`${baseClass}__chart-container`}>{renderChart()}</div>
      {showFilterModal && (
        <ChartFilterModal
          filters={chartFilters}
          currentTeamId={currentTeamId}
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
