import { isEmpty } from "lodash";

import { PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import {
  CVE_SOFTWARE_CATEGORIES,
  ALL_CVE_SOFTWARE_CATEGORY_VALUES,
  IVulnExposureFilterDefaults,
} from "interfaces/charts";
import {
  ANY_SEVERITY_VALUE,
  getSeverityBand,
  ISeverityFilterValue,
  severityFilters,
  severityForRange,
  severityValueLabel,
} from "components/SeverityFilter";

import { IChartFilterState } from "./ChartFilterModal";
import { isEpssActive } from "./ChartFilterModal/SoftwareFilters/helpers";

const DEFAULT_CHART_FILTERS: IChartFilterState = {
  labelIDs: [],
  platforms: [],
  hostFilterMode: "none",
  selectedHosts: [],
  softwareFilters: [...ALL_CVE_SOFTWARE_CATEGORY_VALUES],
  knownExploit: false,
  epssMin: "",
  epssMax: "",
  // The chart is filtered to critical severity by default. The bounds always
  // mirror the selection (see IChartFilterState), so they carry critical's.
  severity: "critical",
  cvssMin: "9",
  cvssMax: "10",
  excludeCVEs: [],
};

// Bounds are materialized for every option except Any, which holds none — the
// same invariant SeverityFilter maintains when the user picks an option.
const seedSeverity = (defaults: IVulnExposureFilterDefaults) => {
  if (defaults.cvss_min === undefined && defaults.cvss_max === undefined) {
    return {
      severity: DEFAULT_CHART_FILTERS.severity,
      cvssMin: DEFAULT_CHART_FILTERS.cvssMin,
      cvssMax: DEFAULT_CHART_FILTERS.cvssMax,
    };
  }
  const severity = severityForRange(defaults.cvss_min, defaults.cvss_max);
  if (severity === ANY_SEVERITY_VALUE) {
    return { severity, cvssMin: "", cvssMax: "" };
  }
  // A Custom range widens a missing bound to that end of the scale. `??`, not
  // `||`: `cvss_max: 0` means "only 0.0", not "up to 10".
  const band = getSeverityBand(severity);
  return {
    severity,
    cvssMin: String(band?.min ?? defaults.cvss_min ?? 0),
    cvssMax: String(band?.max ?? defaults.cvss_max ?? 10),
  };
};

// Seeds from the GitOps-managed defaults, per field: an undefined field falls
// back to DEFAULT_CHART_FILTERS, while a present one is respected — including
// an explicit empty software_filters list, which means "no categories".
export const buildInitialChartFilters = (
  defaults?: IVulnExposureFilterDefaults
): IChartFilterState => {
  if (!defaults) return DEFAULT_CHART_FILTERS;
  return {
    ...DEFAULT_CHART_FILTERS,
    ...seedSeverity(defaults),
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

// The chart state spells the scores cvssMin/cvssMax; SeverityFilter's helpers
// take minScore/maxScore.
export const severitySelection = (
  filters: IChartFilterState
): ISeverityFilterValue => ({
  severity: filters.severity,
  minScore: filters.cvssMin,
  maxScore: filters.cvssMax,
});

export const hasActiveHostFilters = (filters: IChartFilterState): boolean => {
  const hasHostFilter =
    filters.hostFilterMode !== "none" && filters.selectedHosts.length > 0;
  return (
    filters.labelIDs.length > 0 || filters.platforms.length > 0 || hasHostFilter
  );
};

export const hasActiveSoftwareFilters = (filters: IChartFilterState): boolean =>
  filters.softwareFilters.length !== ALL_CVE_SOFTWARE_CATEGORY_VALUES.length ||
  filters.knownExploit ||
  isEpssActive(filters.epssMin, filters.epssMax) ||
  !isEmpty(severityFilters(severitySelection(filters))) ||
  filters.excludeCVEs.length > 0;

// Human-readable "a, b, and c". Items must already be correctly cased —
// don't force-capitalize here or branded names like "macOS"/"iOS" break.
const formatList = (items: string[]): string => {
  if (items.length <= 1) return items.join("");
  if (items.length === 2) return `${items[0]} and ${items[1]}`;
  return `${items.slice(0, -1).join(", ")}, and ${items[items.length - 1]}`;
};

// String-indexable, since a platform filter value is an arbitrary string — an
// unknown one indexes to undefined and falls back to the raw value below.
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

export const softwareFilterLines = (filters: IChartFilterState): string[] => {
  const lines: string[] = [];
  // All categories are selected by default, so an unnarrowed selection is not
  // an active filter and gets no line.
  const categoriesNarrowed =
    filters.softwareFilters.length !== ALL_CVE_SOFTWARE_CATEGORY_VALUES.length;
  const cats = CVE_SOFTWARE_CATEGORIES.filter((c) =>
    filters.softwareFilters.includes(c.value)
  ).map((c) => c.tooltipLabel);
  if (categoriesNarrowed) {
    lines.push(cats.length ? formatList(cats) : "No software categories");
  }
  if (filters.knownExploit) lines.push("Known exploits only");
  const selection = severitySelection(filters);
  if (!isEmpty(severityFilters(selection))) {
    lines.push(`Severity: ${severityValueLabel(selection)}`);
  }
  if (
    isEpssActive(filters.epssMin, filters.epssMax) ||
    filters.excludeCVEs.length > 0
  ) {
    lines.push("Advanced filters");
  }
  return lines;
};
