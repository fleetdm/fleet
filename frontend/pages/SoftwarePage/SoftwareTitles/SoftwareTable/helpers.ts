import { QueryParams, parseQueryValueToNumberOrUndefined } from "utilities/url";
import stringUtils from "utilities/strings/stringUtils";
import { tooltipTextWithLineBreaks } from "utilities/helpers";

export type ISoftwareDropdownFilterVal =
  | "allSoftware"
  | "installableSoftware"
  | "selfServiceSoftware";

export type IHostSoftwareDropdownFilterVal =
  | ISoftwareDropdownFilterVal
  | "vulnerableSoftware";

export const SOFTWARE_TITLES_DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "All software",
    value: "allSoftware",
    helpText: "All software installed on your hosts.",
  },
  {
    disabled: false,
    label: "Available for install",
    value: "installableSoftware",
    helpText: "Software that can be installed on your hosts.",
  },
  {
    disabled: false,
    label: "Self-service",
    value: "selfServiceSoftware",
    helpText: "Software that end users can install from Fleet Desktop.",
  },
];

export const SEVERITY_DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "Any severity",
    value: "any",
    helpText: "CVSS score 0-10",
    minSeverity: undefined,
    maxSeverity: undefined,
  },
  {
    disabled: false,
    label: "Low severity",
    value: "low",
    helpText: "CVSS score 0.1-3.9",
    minSeverity: 0.1,
    maxSeverity: 3.9,
  },
  {
    disabled: false,
    label: "Medium severity",
    value: "medium",
    helpText: "CVSS score 4.0-6.9",
    minSeverity: 4.0,
    maxSeverity: 6.9,
  },
  {
    disabled: false,
    label: "High severity",
    value: "high",
    helpText: "CVSS score 7.0-8.9",
    minSeverity: 7.0,
    maxSeverity: 8.9,
  },
  {
    disabled: false,
    label: "Critical severity",
    value: "critical",
    helpText: "CVSS score 9.0-10",
    minSeverity: 9.0,
    maxSeverity: 10,
  },
];

export const getSoftwareFilterForQueryKey = (
  val: ISoftwareDropdownFilterVal
) => {
  switch (val) {
    case "installableSoftware":
      return { availableForInstall: true };
    case "selfServiceSoftware":
      return { selfService: true };
    default:
      return {};
  }
};

export const getSoftwareFilterFromQueryParams = (queryParams: QueryParams) => {
  const { available_for_install, self_service } = queryParams;
  switch (true) {
    case available_for_install === "true":
      return "installableSoftware";
    case self_service === "true":
      return "selfServiceSoftware";
    default:
      return "allSoftware";
  }
};

export const getSoftwareVulnFiltersFromQueryParams = (
  queryParams: QueryParams
) => {
  const { vulnerable, exploit, min_cvss_score, max_cvss_score } = queryParams;

  return {
    vulnerable: Boolean(vulnerable),
    exploit: Boolean(exploit),
    minCvssScore: parseQueryValueToNumberOrUndefined(min_cvss_score, 0, 10),
    maxCvssScore: parseQueryValueToNumberOrUndefined(max_cvss_score, 0, 10),
  };
};

export type ISoftwareVulnFilters = {
  vulnerable?: boolean;
  exploit?: boolean;
  min_cvss_score?: number;
  max_cvss_score?: number;
};

export type ISoftwareVulnFiltersParams = {
  vulnerable?: boolean;
  exploit?: boolean;
  minCvssScore?: number;
  maxCvssScore?: number;
};

export const getSoftwareVulnFiltersForQueryKey = (
  vulnFilters: ISoftwareVulnFilters
) => {
  const { vulnerable, exploit, min_cvss_score, max_cvss_score } = vulnFilters;

  if (!vulnerable) {
    return {};
  }

  const isValidNumber = (value: any): value is number =>
    value !== null && value !== undefined && !isNaN(value);

  return {
    vulnerable: true,
    ...(exploit && { exploit: true }),
    ...(isValidNumber(min_cvss_score) && { min_cvss_score }),
    ...(isValidNumber(max_cvss_score) && { max_cvss_score }),
  };
};

export const findOptionBySeverityRange = (
  minSeverityValue: number | undefined,
  maxSeverityValue: number | undefined
) => {
  const severityOption = SEVERITY_DROPDOWN_OPTIONS.find(
    (option) =>
      option.minSeverity === minSeverityValue &&
      option.maxSeverity === maxSeverityValue
  ) || {
    disabled: true,
    label: "Custom severity",
    value: "custom",
    helpText: `CVSS score ${minSeverityValue || 0}-${maxSeverityValue || 10}`,
    minSeverity: minSeverityValue || 0,
    maxSeverity: maxSeverityValue || 10,
  };

  return severityOption;
};

export const getVulnFilterRenderDetails = (
  vulnFilters?: ISoftwareVulnFiltersParams
) => {
  let filterCount = 0;
  const tooltipText = [];

  if (vulnFilters) {
    if (vulnFilters.vulnerable) {
      filterCount += 1;
      tooltipText.push("Vulnerable software");

      if (vulnFilters.minCvssScore || vulnFilters.maxCvssScore) {
        filterCount += 1;
        const severityOption = findOptionBySeverityRange(
          vulnFilters.minCvssScore,
          vulnFilters.maxCvssScore
        );
        const severityText = stringUtils.capitalize(severityOption?.value);
        tooltipText.push(`Severity: ${severityText}`);
      }

      if (vulnFilters.exploit) {
        filterCount += 1;
        tooltipText.push("Has known exploit");
      }
    }
  }

  const buttonText =
    filterCount > 0
      ? `${filterCount} filter${filterCount > 1 ? "s" : ""}`
      : "Add filters";

  return {
    filterCount,
    buttonText,
    tooltipText: tooltipTextWithLineBreaks(tooltipText),
  };
};
