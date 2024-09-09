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

export const buildSoftwareFilterQueryParams = (
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
    vulnerable: stringUtils.strToBool(vulnerable as string),
    exploit: stringUtils.strToBool(exploit as string),
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

export const isValidNumber = (
  value: any,
  min?: number,
  max?: number
): value is number => {
  // Check if the value is a number and not NaN
  const isNumber = typeof value === "number" && !isNaN(value);

  // If min or max is provided, check if the number is within the range
  const withinRange =
    (min === undefined || value >= min) && (max === undefined || value <= max);

  return isNumber && withinRange;
};

export const buildSoftwareVulnFiltersQueryParams = (
  vulnFilters: ISoftwareVulnFiltersParams
) => {
  const { vulnerable, exploit, minCvssScore, maxCvssScore } = vulnFilters;

  if (!vulnerable) {
    return {};
  }

  return {
    vulnerable: true,
    ...(exploit && { exploit: true }),
    ...(isValidNumber(minCvssScore, 0, maxCvssScore || 10) && {
      min_cvss_score: minCvssScore.toString(),
    }),
    ...(isValidNumber(maxCvssScore, minCvssScore || 0, 10) && {
      max_cvss_score: maxCvssScore.toString(),
    }),
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
