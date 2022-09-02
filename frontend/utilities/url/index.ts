import { isEmpty, reduce, omitBy, Dictionary } from "lodash";

type QueryValues = string | number | boolean | undefined | null;
export type QueryParams = Record<string, QueryValues>;
type FilteredQueryValues = string | number | boolean;
type FilteredQueryParams = Record<string, FilteredQueryValues>;

const reduceQueryParams = (
  params: string[],
  value: FilteredQueryValues,
  key: string
) => {
  key && params.push(`${encodeURIComponent(key)}=${encodeURIComponent(value)}`);
  return params;
};

const filterEmptyParams = (queryParams: QueryParams) => {
  return omitBy(
    queryParams,
    (value) => value === undefined || value === "" || value === null
  ) as Dictionary<FilteredQueryValues>;
};

/**
 * creates a query string from a query params object. If a value is undefined, null,
 * or an empty string on the queryParams object, that key-value pair will be
 * excluded from the query string.
 */
export const buildQueryStringFromParams = (queryParams: QueryParams) => {
  const filteredParams = filterEmptyParams(queryParams);

  let queryString = "";
  if (!isEmpty(queryParams)) {
    queryString = reduce<FilteredQueryParams, string[]>(
      filteredParams,
      reduceQueryParams,
      []
    ).join("&");
  }
  return queryString;
};

export const reconcileMutuallyExclusiveHostParams = (
  label?: string,
  policyId?: number,
  policyResponse?: string,
  mdmId?: number,
  mdmEnrollmentStatus?: string,
  munkiIssueId?: number,
  softwareId?: number,
  osId?: number,
  osName?: string,
  osVersion?: string
): Record<string, unknown> => {
  if (label) {
    return {};
  }
  switch (true) {
    case !!policyId:
      return { policy_id: policyId, policy_response: policyResponse };
    case !!mdmId:
      return { mdm_id: mdmId, mdm_status: mdmEnrollmentStatus };
    case !!munkiIssueId:
      return { munki_issue_id: munkiIssueId };
    case !!softwareId:
      return { software_id: softwareId };
    case !!osId:
      return { os_id: osId };
    case !!osName && !!osVersion:
      return { os_name: osName, os_version: osVersion };
    default:
      return {};
  }
};

const LABEL_PREFIX = "labels/";

export const getStatusParam = (selectedLabels?: string[]) => {
  if (selectedLabels === undefined) return undefined;

  const status = selectedLabels.find((f) => !f.includes(LABEL_PREFIX));
  if (status === undefined) return undefined;

  const statusFilterList = ["new", "online", "offline"];
  return statusFilterList.includes(status) ? status : undefined;
};

export const getLabelParam = (selectedLabels?: string[]) => {
  if (selectedLabels === undefined) return undefined;

  const label = selectedLabels.find((f) => f.includes(LABEL_PREFIX));
  if (label === undefined) return undefined;

  return label.slice(7);
};
