import { parseSoftwareTypesParam, SOFTWARE_TYPES } from "interfaces/software";
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
    ai_tool,
  } = queryParams;

  return {
    vulnerable: stringUtils.strToBool(vulnerable as string),
    exploit: stringUtils.strToBool(exploit as string),
    minCvssScore: parseQueryValueToNumberOrUndefined(min_cvss_score, 0, 10),
    maxCvssScore: parseQueryValueToNumberOrUndefined(max_cvss_score, 0, 10),
    types: parseSoftwareTypesParam(types as string | undefined),
    aiTool: ai_tool === "true",
  };
};

export type ISoftwareFilters = {
  vulnerable?: boolean;
  exploit?: boolean;
  minCvssScore?: number;
  maxCvssScore?: number;
  types?: string[];
  aiTool?: boolean;
};

/** Page URL params for the Filters modal's selections. */
export type ISoftwareFiltersQueryParams = {
  types?: string;
  vulnerable?: boolean;
  exploit?: boolean;
  min_cvss_score?: string;
  max_cvss_score?: string;
  ai_tool?: boolean;
};

export const buildSoftwareFiltersQueryParams = (
  filters: ISoftwareFilters
): ISoftwareFiltersQueryParams => {
  const {
    vulnerable,
    exploit,
    minCvssScore,
    maxCvssScore,
    types,
    aiTool,
  } = filters;

  const baseParams = {
    ...(types?.length && { types: [...types].sort().join(",") }),
    ...(aiTool && { ai_tool: true }),
  };

  if (!vulnerable) {
    return baseParams;
  }

  return {
    ...baseParams,
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

/** Filter button label: "Add filters" until any type, AI tools or
 * vulnerability filter is applied, then "Filtered" (no count). */
export const getFilterRenderDetails = (filters?: ISoftwareFilters) => {
  const isFiltered =
    !!filters?.vulnerable || !!filters?.types?.length || !!filters?.aiTool;

  return {
    isFiltered,
    buttonText: isFiltered ? "Filtered" : "Add filters",
  };
};

const PREMIUM_ONLY_TYPE_KEYS = new Set(
  SOFTWARE_TYPES.filter((t) => t.premiumOnly).map((t) => t.key)
);

/** Drops filters the API rejects on Fleet Free, which a URL copied from a
 * Premium server can still carry. */
export const removePremiumOnlyFilters = (
  filters: ISoftwareFilters
): ISoftwareFilters => ({
  ...filters,
  aiTool: false,
  types: filters.types?.filter((key) => !PREMIUM_ONLY_TYPE_KEYS.has(key)),
});

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
