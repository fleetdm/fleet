import React, { useContext, useEffect, useState, useMemo } from "react";
import { useQuery } from "react-query";
import { format, parseISO } from "date-fns";
import { isEqual } from "lodash";
import { SingleValue } from "react-select-5";

import chartsAPI, {
  IChartResponse,
  IChartApiParams,
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
import { severityFilters } from "components/SeverityFilter";

import {
  IDataSet,
  IFormattedDataPoint,
  DATASET_CONFIG_KEY,
  DATASET_LABEL,
  HistoricalDataConfigKey,
  ALL_CVE_SOFTWARE_CATEGORY_VALUES,
  IVulnExposureFilterDefaults,
} from "interfaces/charts";

import { AppContext } from "context/app";

import ChartFilterModal, {
  IChartFilterState,
  ChartFilterTab,
} from "./ChartFilterModal";
import {
  buildInitialChartFilters,
  hasActiveHostFilters,
  hasActiveSoftwareFilters,
  hostFilterLines,
  severitySelection,
  softwareFilterLines,
} from "./helpers";
import LineChartViz from "./LineChartViz";
import CheckerboardViz from "./CheckerboardViz";
import DataCollectionDisabledState from "./DataCollectionDisabledState";

const baseClass = "chart-card";

// All charts are currently fixed at a 30-day window. When the server supports
// configurable ranges we'll add UI and request-param plumbing for this.
const CHART_DAYS = 30;

// A single consolidated tooltip summarizing every active filter, grouped into
// "Hosts" and "Software" sections. Each section is omitted when it has no
// active filters; software filters only apply to the cve dataset.
const filterTooltip = (
  filters: IChartFilterState,
  isCVE: boolean
): JSX.Element => {
  const hostLines = hostFilterLines(filters);
  const softwareLines = isCVE ? softwareFilterLines(filters) : [];
  const renderSection = (header: string, lines: string[]) =>
    lines.length > 0 ? (
      <div className={`${baseClass}__tooltip-section`}>
        <div className={`${baseClass}__tooltip-section-header`}>{header}</div>
        {lines.map((line) => (
          <div key={line} className={`${baseClass}__tooltip-section-line`}>
            {line}
          </div>
        ))}
      </div>
    ) : null;
  return (
    <>
      {renderSection("Hosts", hostLines)}
      {renderSection("Software", softwareLines)}
    </>
  );
};

interface IChartCardProps {
  currentTeamId?: number;
  historicalDataEnabled?: Record<HistoricalDataConfigKey, boolean>;
  // GitOps-managed default filter state for the current scope (org or fleet).
  // Seeds the chart's filter controls on load; UI edits are not persisted.
  filterDefaults?: IVulnExposureFilterDefaults;
}

const ChartCard = ({
  currentTeamId,
  historicalDataEnabled,
  filterDefaults,
}: IChartCardProps): JSX.Element => {
  const [selectedMetric, setSelectedMetric] = useState("uptime");
  const [showFilterModal, setShowFilterModal] = useState(false);
  const [initialTab, setInitialTab] = useState<ChartFilterTab>("hosts");
  const [showAdvancedOnOpen, setShowAdvancedOnOpen] = useState(false);
  // The chart's baseline. Nothing in it was chosen by the user, so the
  // "Filtered" pill keys on differing from it rather than on filters being set.
  const initialChartFilters = useMemo(
    () => buildInitialChartFilters(filterDefaults),
    [filterDefaults]
  );
  const [chartFilters, setChartFilters] = useState<IChartFilterState>(
    initialChartFilters
  );

  const openFilterModal = (
    tab: ChartFilterTab = "hosts",
    showAdvanced = false
  ) => {
    setInitialTab(tab);
    setShowAdvancedOnOpen(showAdvanced);
    setShowFilterModal(true);
  };

  const { isPremiumTier } = useContext(AppContext);

  const DATASETS: IDataSet[] = [
    {
      name: "uptime",
      label: "Hosts online",
      defaultChartType: "checkerboard",
      description: (
        <>
          The number of hosts detected online (checking in to Fleet) during a
          given hour.
          <br />
          <br />
          iOS/iPadOS hosts are online anytime they have power and an internet
          connection (including locked). macOS, Windows, and Linux hosts can be
          online when locked (lid closed), but less frequently than when the lid
          is open. Android hosts are never online when locked.
        </>
      ),
      tooltipFormatter: ({ value }: { value: number }) =>
        `${value.toLocaleString()} host${value === 1 ? "" : "s"} online`,
      relativeScale: true,
    },
  ];

  const getDataset = (name: string): IDataSet =>
    DATASETS.find((ds) => ds.name === name) || DATASETS[0];

  if (isPremiumTier) {
    DATASETS.push({
      name: "cve",
      label: "Vulnerability exposure",
      defaultChartType: "checkerboard",
      description: (
        <>
          The number of hosts with at least one vulnerability matching the
          chart&apos;s filters.
          <br />
          <br />
          Severity is filtered to critical by default.
        </>
      ),
      tooltipFormatter: ({ value }: { value: number }) =>
        `${value.toLocaleString()} host${value === 1 ? "" : "s"}`,
      theme: "red",
      relativeScale: true,
    });
  }

  const DATASET_OPTIONS: CustomOptionType[] = DATASETS.map((ds) => ({
    label: ds.label,
    value: ds.name,
  }));

  // Labels and selected hosts are team-scoped, so clear filters when the
  // active fleet changes to avoid submitting stale IDs under the new scope.
  // Re-seed from the persisted defaults when the scope changes (fleet switch)
  // or once the config/fleet data finishes loading. This also discards any
  // ephemeral UI edits, matching the "UI edits are not saved" behavior.
  useEffect(() => {
    setChartFilters(initialChartFilters);
  }, [currentTeamId, initialChartFilters]);

  const currentDataset = getDataset(selectedMetric);

  const isCVE = currentDataset.name === "cve";
  const hostFiltersActive = hasActiveHostFilters(chartFilters);
  const softwareFiltersActive = isCVE && hasActiveSoftwareFilters(chartFilters);
  // A seeded default is how the chart always looks, so flagging it as
  // "Filtered" on load would say nothing. Once shown, the tooltip lists
  // everything narrowing the data, defaults included.
  const filtersEdited = !isEqual(chartFilters, initialChartFilters);
  const anyFiltersActive =
    filtersEdited && (hostFiltersActive || softwareFiltersActive);

  const datasetConfigKey = DATASET_CONFIG_KEY[currentDataset.name];
  // If a dataset has no config-key mapping (future addition), treat it as
  // enabled — collection toggles only apply to known config keys.
  const datasetCollectionEnabled =
    datasetConfigKey === undefined
      ? true
      : historicalDataEnabled?.[datasetConfigKey] ?? true;

  const queryParams: IChartApiParams = useMemo(() => {
    // Only narrow categories when not all are selected; EPSS only narrows when
    // min > 0 or max < 100. The Software tab enters EPSS as 0–100 %, but the
    // API takes 0.0–1.0, so divide before sending.
    const narrowsCategories =
      isCVE &&
      chartFilters.softwareFilters.length !==
        ALL_CVE_SOFTWARE_CATEGORY_VALUES.length;
    const epssMinActive =
      isCVE && chartFilters.epssMin !== "" && Number(chartFilters.epssMin) > 0;
    const epssMaxActive =
      isCVE &&
      chartFilters.epssMax !== "" &&
      Number(chartFilters.epssMax) < 100;

    // filterEmptyParams drops undefined/""/null, so a legitimate 0 survives.
    const severityBounds = isCVE
      ? severityFilters(severitySelection(chartFilters))
      : {};

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
      software_filters: narrowsCategories
        ? chartFilters.softwareFilters.join(",")
        : undefined,
      has_known_exploit: isCVE && chartFilters.knownExploit ? true : undefined,
      epss_min: epssMinActive ? Number(chartFilters.epssMin) / 100 : undefined,
      epss_max: epssMaxActive ? Number(chartFilters.epssMax) / 100 : undefined,
      severity_min: severityBounds.min,
      severity_max: severityBounds.max,
      exclude_vulnerabilities:
        isCVE && chartFilters.excludeCVEs.length
          ? chartFilters.excludeCVEs.join(",")
          : undefined,
    };
  }, [chartFilters, currentTeamId, isCVE]);

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
      return <Spinner verticalPadding="small" />;
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
          {anyFiltersActive && (
            <TooltipWrapper
              tipContent={filterTooltip(chartFilters, isCVE)}
              position="top"
              underline={false}
              showArrow
              tipOffset={8}
            >
              <button
                type="button"
                className={`${baseClass}__filter-pill`}
                onClick={() =>
                  openFilterModal(
                    hostFiltersActive ? "hosts" : "software",
                    true
                  )
                }
              >
                Filtered
              </button>
            </TooltipWrapper>
          )}
        </div>
        <div className={`${baseClass}__header-right`}>
          <Button
            type="button"
            variant="subdued"
            size="small"
            ariaLabel="Configure chart filters"
            onClick={() => openFilterModal()}
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
          metric={selectedMetric}
          initialTab={initialTab}
          initialShowAdvanced={showAdvancedOnOpen}
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
