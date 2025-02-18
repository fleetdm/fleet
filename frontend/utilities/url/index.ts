import { isEmpty, reduce, omitBy, Dictionary, snakeCase } from "lodash";

import {
  DiskEncryptionStatus,
  BootstrapPackageStatus,
  MdmProfileStatus,
} from "interfaces/mdm";
import {
  HOSTS_QUERY_PARAMS,
  MacSettingsStatusQueryParam,
} from "services/entities/hosts";
import { isValidSoftwareAggregateStatus } from "interfaces/software";
import { API_ALL_TEAMS_ID } from "interfaces/team";

export type QueryValues = string | number | boolean | undefined | null;
export type QueryParams = Record<string, QueryValues>;
/** updated value for query params. TODO: update using this value across the codebase */
type QueryParams2<T> = { [s in keyof T]: QueryValues };
type FilteredQueryValues = string | number | boolean;
type FilteredQueryParams = Record<string, FilteredQueryValues>;

interface IMutuallyInclusiveHostParams {
  label?: string;
  teamId?: number;
  macSettingsStatus?: MacSettingsStatusQueryParam;
  osSettings?: MdmProfileStatus;
}

interface IMutuallyExclusiveHostParams {
  teamId?: number;
  label?: string;
  policyId?: number;
  policyResponse?: string;
  mdmId?: number;
  mdmEnrollmentStatus?: string;
  munkiIssueId?: number;
  lowDiskSpaceHosts?: number;
  softwareId?: number;
  softwareVersionId?: number;
  softwareTitleId?: number;
  softwareStatus?: string;
  osVersionId?: number;
  osName?: string;
  osVersion?: string;
  vulnerability?: string;
  osSettings?: MdmProfileStatus;
  diskEncryptionStatus?: DiskEncryptionStatus;
  bootstrapPackageStatus?: BootstrapPackageStatus;
}

export const parseQueryValueToNumberOrUndefined = (
  value: QueryValues,
  min?: number,
  max?: number
): number | undefined => {
  const isWithinRange = (num: number) => {
    if (min !== undefined && max !== undefined) {
      return num >= min && num <= max;
    }
    return true; // No range check if min or max is undefined
  };

  if (typeof value === "number") {
    return isWithinRange(value) ? value : undefined;
  }
  if (typeof value === "string") {
    const parsedValue = parseFloat(value);
    return !isNaN(parsedValue) && isWithinRange(parsedValue)
      ? parsedValue
      : undefined;
  }
  return undefined;
};

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
export const buildQueryStringFromParams = <T>(queryParams: QueryParams2<T>) => {
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

export const reconcileSoftwareParams = ({
  teamId,
  softwareId,
  softwareVersionId,
  softwareTitleId,
  softwareStatus,
}: Pick<
  IMutuallyExclusiveHostParams,
  | "teamId"
  | "softwareId"
  | "softwareVersionId"
  | "softwareTitleId"
  | "softwareStatus"
>) => {
  if (
    isValidSoftwareAggregateStatus(softwareStatus) &&
    softwareTitleId &&
    teamId !== API_ALL_TEAMS_ID
  ) {
    return {
      software_title_id: softwareTitleId,
      [HOSTS_QUERY_PARAMS.SOFTWARE_STATUS]: softwareStatus,
      team_id: teamId,
    };
  }

  if (softwareTitleId) {
    return { software_title_id: softwareTitleId };
  }

  if (softwareVersionId) {
    return { software_version_id: softwareVersionId };
  }

  if (softwareId) {
    return { software_id: softwareId };
  }

  return {};
};

export const reconcileMutuallyInclusiveHostParams = ({
  label,
  teamId,
  macSettingsStatus,
  osSettings,
}: IMutuallyInclusiveHostParams) => {
  const reconciled: Record<string, unknown> = { team_id: teamId };

  if (label) {
    // if label is present, include team_id in the query but exclude others
    return reconciled;
  }

  if (macSettingsStatus) {
    // ensure macos_settings filter is always applied in
    // conjuction with a team_id, 0 (no teams) by default
    reconciled.macos_settings = macSettingsStatus;
    reconciled.team_id = teamId ?? 0;
  }
  if (osSettings) {
    // ensure os_settings filter is always applied in
    // conjuction with a team_id, 0 (no teams) by default
    reconciled[HOSTS_QUERY_PARAMS.OS_SETTINGS] = osSettings;
    reconciled.team_id = teamId ?? 0;
  }

  return reconciled;
};

export const reconcileMutuallyExclusiveHostParams = ({
  teamId,
  label,
  policyId,
  policyResponse,
  mdmId,
  mdmEnrollmentStatus,
  munkiIssueId,
  lowDiskSpaceHosts,
  softwareId,
  softwareVersionId,
  softwareTitleId,
  softwareStatus,
  osVersionId,
  osName,
  osVersion,
  osSettings,
  vulnerability,
  diskEncryptionStatus,
  bootstrapPackageStatus,
}: IMutuallyExclusiveHostParams): Record<string, unknown> => {
  if (label) {
    // backend api now allows (label + low disk space) OR (label + mdm id) OR
    // (label + mdm enrollment status). all other params are still mutually exclusive.
    if (mdmId) {
      return { mdm_id: mdmId };
    }
    if (mdmEnrollmentStatus) {
      return { mdm_enrollment_status: mdmEnrollmentStatus };
    }
    if (lowDiskSpaceHosts) {
      return { low_disk_space: lowDiskSpaceHosts };
    }
    return {};
  }

  switch (true) {
    case !!policyId:
      return { policy_id: policyId, policy_response: policyResponse };
    case !!mdmId:
      return { mdm_id: mdmId };
    case !!mdmEnrollmentStatus:
      return { mdm_enrollment_status: mdmEnrollmentStatus };
    case !!munkiIssueId:
      return { munki_issue_id: munkiIssueId };
    case !!softwareStatus ||
      !!softwareTitleId ||
      !!softwareVersionId ||
      !!softwareId:
      return reconcileSoftwareParams({
        teamId,
        softwareId,
        softwareVersionId,
        softwareTitleId,
        softwareStatus,
      });
    case !!softwareVersionId:
      return { software_version_id: softwareVersionId };
    case !!softwareId:
      return { software_id: softwareId };
    case !!osVersionId:
      return { os_version_id: osVersionId };
    case !!osName && !!osVersion:
      return { os_name: osName, os_version: osVersion };
    case !!vulnerability:
      return { vulnerability };
    case !!lowDiskSpaceHosts:
      return { low_disk_space: lowDiskSpaceHosts };
    case !!osSettings:
      return { [HOSTS_QUERY_PARAMS.OS_SETTINGS]: osSettings };
    case !!diskEncryptionStatus:
      return { [HOSTS_QUERY_PARAMS.DISK_ENCRYPTION]: diskEncryptionStatus };
    case !!bootstrapPackageStatus:
      return { bootstrap_package: bootstrapPackageStatus };
    default:
      return {};
  }
};

const LABEL_PREFIX = "labels/";

export const getStatusParam = (selectedLabels?: string[]) => {
  if (selectedLabels === undefined) return undefined;

  const status = selectedLabels.find((f) => !f.includes(LABEL_PREFIX));
  if (status === undefined) return undefined;

  const statusFilterList = ["new", "online", "offline", "missing"];
  return statusFilterList.includes(status) ? status : undefined;
};

export const getLabelParam = (selectedLabels?: string[]) => {
  if (selectedLabels === undefined) return undefined;

  const label = selectedLabels.find((f) => f.includes(LABEL_PREFIX));
  if (label === undefined) return undefined;

  return label.slice(7);
};

type QueryParamish<T> = keyof T extends string
  ? {
      [K in keyof T]: QueryValues;
    }
  : never;

export const convertParamsToSnakeCase = <T extends QueryParamish<T>>(
  params: T
) => {
  return reduce<typeof params, QueryParams>(
    params,
    (result, val, key) => {
      result[snakeCase(key)] = val;
      return result;
    },
    {}
  );
};
