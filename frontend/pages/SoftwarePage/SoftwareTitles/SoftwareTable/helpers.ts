import { QueryParams, parseQueryValueToNumberOrUndefined } from "utilities/url";

export type ISoftwareDropdownFilterVal =
  | "allSoftware"
  | "installableSoftware"
  | "selfServiceSoftware";

export type IHostSoftwareDropdownFilterVal =
  | ISoftwareDropdownFilterVal
  | "vulnerableSoftware";

const ALL_SOFTWARE_OPTION = {
  disabled: false,
  label: "All software",
  value: "allSoftware",
  helpText: "All software installed on your hosts.",
};

const INSTALLABLE_SOFTWARE_OPTION = {
  disabled: false,
  label: "Available for install",
  value: "installableSoftware",
  helpText: "Software that can be installed on your hosts.",
};

const SELF_SERVICE_SOFTWARE_OPTION = {
  disabled: false,
  label: "Self-service",
  value: "selfServiceSoftware",
  helpText: "Software that end users can install from Fleet Desktop.",
};

export const SOFTWARE_VERSIONS_DROPDOWN_OPTIONS = [ALL_SOFTWARE_OPTION];

export const SOFTWARE_TITLES_DROPDOWN_OPTIONS = [
  ALL_SOFTWARE_OPTION,
  INSTALLABLE_SOFTWARE_OPTION,
  SELF_SERVICE_SOFTWARE_OPTION,
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
    minCvssScore: parseQueryValueToNumberOrUndefined(min_cvss_score),
    maxCvssScore: parseQueryValueToNumberOrUndefined(max_cvss_score),
  };
};

export type ISoftwareVulnFilters = {
  vulnerable?: boolean;
  exploit?: boolean;
  min_cvss_score?: number;
  max_cvss_score?: number;
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
