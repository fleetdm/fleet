import { isEmpty } from "lodash";

import { QueryParams, parseQueryValueToNumberOrUndefined } from "utilities/url";
import stringUtils from "utilities/strings/stringUtils";
import { tooltipTextWithLineBreaks } from "utilities/helpers";
import numberUtils from "utilities/numbers";

import {
  ISeverityFilterValue,
  severityFilters,
  severityForRange,
  severityValueLabel,
} from "components/SeverityFilter";

const { isValidNumber } = numberUtils;

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

export const getVulnFilterRenderDetails = (
  vulnFilters?: ISoftwareVulnFiltersParams
) => {
  let filterCount = 0;
  const tooltipText = [];

  if (vulnFilters) {
    if (vulnFilters.vulnerable) {
      filterCount += 1;
      tooltipText.push("Vulnerable software");

      const severity: ISeverityFilterValue = {
        severity: severityForRange(
          vulnFilters.minCvssScore,
          vulnFilters.maxCvssScore
        ),
        minScore: vulnFilters.minCvssScore?.toString() ?? "",
        maxScore: vulnFilters.maxCvssScore?.toString() ?? "",
      };
      if (!isEmpty(severityFilters(severity))) {
        filterCount += 1;
        tooltipText.push(`Severity: ${severityValueLabel(severity)}`);
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
