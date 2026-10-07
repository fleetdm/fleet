import { isEmpty } from "lodash";

import {
  ISeverityFilterValue,
  severityFilters,
  severityForRange,
  severityValueLabel,
} from "components/SeverityFilter";
import { parseSoftwareTypesParam } from "interfaces/software";
import { tooltipTextWithLineBreaks } from "utilities/helpers";
import numberUtils from "utilities/numbers";
import stringUtils from "utilities/strings/stringUtils";
import { QueryParams, parseQueryValueToNumberOrUndefined } from "utilities/url";

const { isValidNumber } = numberUtils;

export const getSoftwareFiltersFromQueryParams = (queryParams: QueryParams) => {
  const {
    vulnerable,
    exploit,
    min_cvss_score,
    max_cvss_score,
    types,
  } = queryParams;

  return {
    vulnerable: stringUtils.strToBool(vulnerable as string),
    exploit: stringUtils.strToBool(exploit as string),
    minCvssScore: parseQueryValueToNumberOrUndefined(min_cvss_score, 0, 10),
    maxCvssScore: parseQueryValueToNumberOrUndefined(max_cvss_score, 0, 10),
    types: parseSoftwareTypesParam(types as string | undefined),
  };
};

export type ISoftwareFilters = {
  vulnerable?: boolean;
  exploit?: boolean;
  minCvssScore?: number;
  maxCvssScore?: number;
  types?: string[];
};

/** Page URL params for the Filters modal's selections. */
export type ISoftwareFiltersQueryParams = {
  types?: string;
  vulnerable?: boolean;
  exploit?: boolean;
  min_cvss_score?: string;
  max_cvss_score?: string;
};

export const buildSoftwareFiltersQueryParams = (
  filters: ISoftwareFilters
): ISoftwareFiltersQueryParams => {
  const { vulnerable, exploit, minCvssScore, maxCvssScore, types } = filters;

  const typesParam = types?.length
    ? { types: [...types].sort().join(",") }
    : {};

  if (!vulnerable) {
    return typesParam;
  }

  return {
    ...typesParam,
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

export const getVulnFilterRenderDetails = (filters?: ISoftwareFilters) => {
  let filterCount = 0;
  const tooltipText = [];

  if (filters) {
    if (filters.vulnerable) {
      filterCount += 1;
      tooltipText.push("Vulnerable software");

      const severity: ISeverityFilterValue = {
        severity: severityForRange(filters.minCvssScore, filters.maxCvssScore),
        minScore: filters.minCvssScore?.toString() ?? "",
        maxScore: filters.maxCvssScore?.toString() ?? "",
      };
      if (!isEmpty(severityFilters(severity))) {
        filterCount += 1;
        tooltipText.push(`Severity: ${severityValueLabel(severity)}`);
      }

      if (filters.exploit) {
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

/** Filter button label: "Add filters" until any type or vulnerability filter
 * is applied, then "Filtered" (no count). */
export const getFilterRenderDetails = (filters?: ISoftwareFilters) => {
  const isFiltered =
    getVulnFilterRenderDetails(filters).filterCount > 0 ||
    !!filters?.types?.length;

  return {
    isFiltered,
    buttonText: isFiltered ? "Filtered" : "Add filters",
  };
};

export const getVulnerabilities = <
  T extends { vulnerabilities: string[] | null }
>(
  versions: T[]
): string[] => {
  if (!versions) {
    return [];
  }

  const vulnerabilities = versions.reduce((acc, current) => {
    if (current.vulnerabilities?.length) {
      current.vulnerabilities.forEach((vuln) => acc.add(vuln));
    }
    return acc;
  }, new Set<string>());

  return [...vulnerabilities];
};
