import React, { useContext, useEffect, useState, useMemo } from "react";
import { useQuery } from "react-query";
import { format, parseISO } from "date-fns";
import { SingleValue } from "react-select-5";

import chartsAPI, {
  IChartResponse,
  IChartApiParams,
  IChartQueryKey,
} from "services/entities/charts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { PLATFORM_DISPLAY_NAMES } from "interfaces/platform";

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
  CVE_SOFTWARE_CATEGORIES,
  ALL_CVE_SOFTWARE_CATEGORY_VALUES,
  IVulnExposureFilterDefaults,
} from "interfaces/charts";

import { AppContext } from "context/app";

import ChartFilterModal, {
  IChartFilterState,
  ChartFilterTab,
} from "./ChartFilterModal";
import { isEpssActive } from "./ChartFilterModal/SoftwareFilters/helpers";
import LineChartViz from "./LineChartViz";
import CheckerboardViz from "./CheckerboardViz";
import DataCollectionDisabledState from "./DataCollectionDisabledState";

const baseClass = "chart-card";

// All charts are currently fixed at a 30-day window. When the server supports
// configurable ranges we'll add UI and request-param plumbing for this.
const CHART_DAYS = 30;

const DEFAULT_CHART_FILTERS: IChartFilterState = {
  labelIDs: [],
  platforms: [],
  hostFilterMode: "none",
  selectedHosts: [],
  softwareFilters: [...ALL_CVE_SOFTWARE_CATEGORY_VALUES],
  knownExploit: false,
  epssMin: "",
  epssMax: "",
  excludeCVEs: [],
};

// Seed the chart's initial filter state from the persisted, GitOps-managed
// defaults. Sparse/per-field: an undefined field falls back to the built-in
// DEFAULT_CHART_FILTERS value, while a present field (including an explicit
// empty software_filters list, meaning "no categories") is respected. EPSS
// bounds are numbers (0–100) in the config and strings in the filter state.
// cvss_min/cvss_max are intentionally NOT wired — there is no severity control
// yet (#47326).
export const buildInitialChartFilters = (
  defaults?: IVulnExposureFilterDefaults
): IChartFilterState => {
  if (!defaults) return DEFAULT_CHART_FILTERS;
  return {
    ...DEFAULT_CHART_FILTERS,
    softwareFilters:
      defaults.software_filters !== undefined
        ? [...defaults.software_filters]
        : DEFAULT_CHART_FILTERS.softwareFilters,
    knownExploit:
      defaults.has_known_exploit !== undefined
        ? defaults.has_known_exploit
        : DEFAULT_CHART_FILTERS.knownExploit,
    epssMin:
      defaults.epss_min !== undefined
        ? String(defaults.epss_min)
        : DEFAULT_CHART_FILTERS.epssMin,
    epssMax:
      defaults.epss_max !== undefined
        ? String(defaults.epss_max)
        : DEFAULT_CHART_FILTERS.epssMax,
    excludeCVEs:
      defaults.exclude_vulnerabilities !== undefined
        ? [...defaults.exclude_vulnerabilities]
        : DEFAULT_CHART_FILTERS.excludeCVEs,
  };
};

const hasActiveHostFilters = (filters: IChartFilterState): boolean => {
  const hasHostFilter =
    filters.hostFilterMode !== "none" && filters.selectedHosts.length > 0;
  return (
    filters.labelIDs.length > 0 || filters.platforms.length > 0 || hasHostFilter
  );
};

const hasActiveSoftwareFilters = (filters: IChartFilterState): boolean =>
  filters.softwareFilters.length !== ALL_CVE_SOFTWARE_CATEGORY_VALUES.length ||
  filters.knownExploit ||
  isEpssActive(filters.epssMin, filters.epssMax) ||
  filters.excludeCVEs.length > 0;

// Human-readable "a, b, and c". Items must already be correctly cased —
// don't force-capitalize here or branded names like "macOS"/"iOS" break.
const formatList = (items: string[]): string => {
  if (items.length <= 1) return items.join("");
  if (items.length === 2) return `${items[0]} and ${items[1]}`;
  return `${items.slice(0, -1).join(", ")}, and ${items[items.length - 1]}`;
};

// A string-indexable view of the display-name map. Platform filter values are
// arbitrary strings, so an unknown one indexes to undefined and we fall back to
// the raw value below.
const PLATFORM_LABELS: Record<string, string> = PLATFORM_DISPLAY_NAMES;

export const hostFilterLines = (filters: IChartFilterState): string[] => {
  const lines: string[] = [];
  if (filters.platforms.length > 0) {
    lines.push(
      formatList(filters.platforms.map((p) => PLATFORM_LABELS[p] ?? p))
    );
  }
  if (filters.labelIDs.length > 0) lines.push("Labels");
  if (
    filters.hostFilterMode === "include" &&
    filters.selectedHosts.length > 0
  ) {
    lines.push("Specific hosts");
  }
  if (
    filters.hostFilterMode === "exclude" &&
    filters.selectedHosts.length > 0
  ) {
    lines.push("Excluded hosts");
  }
  return lines;
};

const softwareFilterLines = (filters: IChartFilterState): string[] => {
  const lines: string[] = [];
  // Only surface category text when the user has actually narrowed the
  // selection — all categories are selected by default, so an unnarrowed
  // selection isn't an active filter and shouldn't show a Software section.
  const categoriesNarrowed =
    filters.softwareFilters.length !== ALL_CVE_SOFTWARE_CATEGORY_VALUES.length;
  const cats = CVE_SOFTWARE_CATEGORIES.filter((c) =>
    filters.softwareFilters.includes(c.value)
  ).map((c) => c.tooltipLabel);
  if (categoriesNarrowed) {
    lines.push(cats.length ? formatList(cats) : "No software categories");
  }
  if (filters.knownExploit) lines.push("Known exploits only");
  if (
    isEpssActive(filters.epssMin, filters.epssMax) ||
    filters.excludeCVEs.length > 0
  ) {
    lines.push("Advanced filters");
  }
  return lines;
};

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
  const [chartFilters, setChartFilters] = useState<IChartFilterState>(() =>
    buildInitialChartFilters(filterDefaults)
  );

  const openFilterModal = (tab: ChartFilterTab = "hosts") => {
    setInitialTab(tab);
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
          iPhones/iPads check in and count as online anytime they have power and
          an internet connection (including locked). Macs count as online
          sometimes (infrequently) when the lid is closed. Android hosts never
          show online when locked.
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
          All critical vulnerabilities.
          <br />
          <br />
          Want more control? Severity (CVSS) filter is{" "}
          <CustomLink
            newTab
            text="coming soon "
            variant="tooltip-link"
            url="https://github.com/fleetdm/fleet/issues/47326"
          />
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
    setChartFilters(buildInitialChartFilters(filterDefaults));
  }, [currentTeamId, filterDefaults]);

  const currentDataset = getDataset(selectedMetric);

  const isCVE = currentDataset.name === "cve";
  const hostFiltersActive = hasActiveHostFilters(chartFilters);
  const softwareFiltersActive = isCVE && hasActiveSoftwareFilters(chartFilters);
  const anyFiltersActive = hostFiltersActive || softwareFiltersActive;

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
                  openFilterModal(hostFiltersActive ? "hosts" : "software")
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
            variant="inverse"
            className={`${baseClass}__settings-btn`}
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
