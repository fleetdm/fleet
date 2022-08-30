/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { IHost } from "interfaces/host";
import {
  buildQueryStringFromParams,
  getLabelParam,
  reconcileMutuallyExclusiveHostParams,
  getStatusParam,
} from "utilities/url";

export interface ISortOption {
  key: string;
  direction: string;
}

export interface ILoadHostsOptions {
  page?: number;
  perPage?: number;
  selectedLabels?: string[];
  globalFilter?: string;
  sortBy?: ISortOption[];
  teamId?: number;
  policyId?: number;
  policyResponse?: string;
  softwareId?: number;
  mdmId?: number;
  mdmEnrollmentStatus?: string;
  osId?: number;
  osName?: string;
  osVersion?: string;
  munkiIssueId?: number;
  device_mapping?: boolean;
  columns?: string;
  visibleColumns?: string;
}

export interface IExportHostsOptions {
  sortBy: ISortOption[];
  page?: number;
  perPage?: number;
  selectedLabels?: string[];
  globalFilter?: string;
  teamId?: number;
  policyId?: number;
  policyResponse?: string;
  softwareId?: number;
  mdmId?: number;
  munkiIssueId?: number;
  mdmEnrollmentStatus?: string;
  osId?: number;
  osName?: string;
  osVersion?: string;
  device_mapping?: boolean;
  columns?: string;
  visibleColumns?: string;
}

export type ILoadHostDetailsExtension = "device_mapping" | "macadmins";

const LABEL_PREFIX = "labels/";

const getLabel = (selectedLabels?: string[]) => {
  if (selectedLabels === undefined) return undefined;
  return selectedLabels.find((filter) => filter.includes(LABEL_PREFIX));
};

const getHostEndpoint = (selectedLabels?: string[]) => {
  const { HOSTS, LABEL_HOSTS } = endpoints;
  if (selectedLabels === undefined) return HOSTS;

  const label = getLabel(selectedLabels);
  if (label) {
    const labelId = label.substr(LABEL_PREFIX.length);
    return LABEL_HOSTS(parseInt(labelId, 10));
  }

  return HOSTS;
};

const getSortParams = (sortOptions?: ISortOption[]) => {
  if (sortOptions === undefined || sortOptions.length === 0) {
    return {};
  }

  const sortItem = sortOptions[0];
  return {
    order_key: sortItem.key,
    order_direction: sortItem.direction,
  };
};

<<<<<<< HEAD
=======
const getStatusParam = (selectedLabels?: string[]) => {
  if (selectedLabels === undefined) return undefined;

  const status = selectedLabels.find((f) => !f.includes(LABEL_PREFIX));
  if (status === undefined) return undefined;

  const statusFilterList = ["new", "online", "offline"];
  return statusFilterList.includes(status) ? status : undefined;
};

const getPolicyParams = (
  label?: string,
  policyId?: number,
  policyResponse?: string
) => {
  if (label !== undefined || policyId === undefined) return {};

  return {
    policy_id: policyId,
    policy_response: policyResponse,
  };
};

const getSoftwareParam = (
  label?: string,
  policyId?: number,
  softwareId?: number,
  mdmId?: number,
  mdmEnrollmentStatus?: string,
  munkiIssueId?: number
) => {
  return !label && !policyId && !mdmId && !mdmEnrollmentStatus && !munkiIssueId
    ? softwareId
    : undefined;
};

const getMDMSolutionParam = (
  label?: string,
  policyId?: number,
  softwareId?: number,
  mdmId?: number,
  mdmEnrollmentStatus?: string,
  munkiIssueId?: number
) => {
  return !label &&
    !policyId &&
    !softwareId &&
    !mdmEnrollmentStatus &&
    !munkiIssueId
    ? mdmId
    : undefined;
};

const getMDMEnrollmentStatusParam = (
  label?: string,
  policyId?: number,
  softwareId?: number,
  mdmId?: number,
  mdmEnrollmentStatus?: string,
  munkiIssueId?: number
) => {
  return !label && !policyId && !softwareId && !mdmId && !munkiIssueId
    ? mdmEnrollmentStatus
    : undefined;
};

const getOperatingSystemParams = (
  label?: string,
  policyId?: number,
  softwareId?: number,
  mdmId?: number,
  mdmEnrollmentStatus?: string,
  munkiIssueId?: number,
  os_id?: number,
  os_name?: string,
  os_version?: string
) => {
  if (
    label ||
    policyId ||
    softwareId ||
    mdmId ||
    mdmEnrollmentStatus ||
    munkiIssueId
  ) {
    return {};
  }
  if (os_id) {
    return { os_id };
  }
  return os_name && os_version ? { os_name, os_version } : {};
};

>>>>>>> 06db6a003 (Fix munki_issue_id to munki_issue, fix pagination)
export default {
  destroy: (host: IHost) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/${host.id}`;

    return sendRequest("DELETE", path);
  },
  destroyBulk: (hostIds: number[]) => {
    const { HOSTS_DELETE } = endpoints;

    return sendRequest("POST", HOSTS_DELETE, { ids: hostIds });
  },
  destroyByFilter: (
    teamId: number | null,
    query: string,
    status: string,
    labelId: number | null
  ) => {
    const { HOSTS_DELETE } = endpoints;
    return sendRequest("POST", HOSTS_DELETE, {
      filters: {
        query,
        status,
        label_id: labelId,
        team_id: teamId,
      },
    });
  },
  exportHosts: (options: IExportHostsOptions) => {
    const sortBy = options.sortBy;
    const selectedLabels = options?.selectedLabels || [];
    const globalFilter = options?.globalFilter || "";
    const teamId = options?.teamId;
    const policyId = options?.policyId;
    const policyResponse = options?.policyResponse || "passing";
<<<<<<< HEAD
    const softwareId = options?.softwareId;
    const mdmId = options?.mdmId;
    const mdmEnrollmentStatus = options?.mdmEnrollmentStatus;
    const visibleColumns = options?.visibleColumns;
    const label = getLabelParam(selectedLabels);
=======
    const softwareId = options?.softwareId || null;
    const mdmId = options?.mdmId || null;
    const mdmEnrollmentStatus = options?.mdmEnrollmentStatus || null;
    const munkiIssueId = options?.munkiIssueId || null;
    const visibleColumns = options?.visibleColumns || null;
    const { os_id, os_name, os_version } = options;
>>>>>>> 06db6a003 (Fix munki_issue_id to munki_issue, fix pagination)

    if (!sortBy.length) {
      throw Error("sortBy is a required field.");
    }

<<<<<<< HEAD
    const queryParams = {
      order_key: sortBy[0].key,
      order_direction: sortBy[0].direction,
      query: globalFilter,
      team_id: teamId,
      ...reconcileMutuallyExclusiveHostParams(
        label,
        policyId,
        policyResponse,
        mdmId,
        mdmEnrollmentStatus,
        softwareId
      ),
      status: getStatusParam(selectedLabels),
      label_id: label,
      columns: visibleColumns,
      format: "csv",
    };
=======
    const orderKeyParam = `?order_key=${sortBy[0].key}`;
    const orderDirection = `&order_direction=${sortBy[0].direction}`;

    let path = `${HOSTS_REPORT}${orderKeyParam}${orderDirection}`;

    if (globalFilter !== "") {
      path += `&query=${globalFilter}`;
    }
    const labelPrefix = "labels/";

    // Handle multiple filters
    const label = selectedLabels.find((f) => f.includes(labelPrefix)) || "";
    const status = selectedLabels.find((f) => !f.includes(labelPrefix)) || "";
    const statusFilterList = ["new", "online", "offline"];
    const isStatusFilter = statusFilterList.includes(status);

    if (isStatusFilter) {
      path += `&status=${status}`;
    }

    if (teamId) {
      path += `&team_id=${teamId}`;
    }

    // label OR policy_id OR software_id OR mdm_id OR mdm_enrollment_status are valid filters.
    if (label) {
      const lid = label.substr(labelPrefix.length);
      path += `&label_id=${parseInt(lid, 10)}`;
    }

    if (!label && policyId) {
      path += `&policy_id=${policyId}`;
      path += `&policy_response=${policyResponse}`;
    }

    if (
      !label &&
      !policyId &&
      !mdmId &&
      !mdmEnrollmentStatus &&
      !munkiIssueId &&
      softwareId
    ) {
      path += `&software_id=${softwareId}`;
    }

    if (
      !label &&
      !policyId &&
      !softwareId &&
      !mdmEnrollmentStatus &&
      !munkiIssueId &&
      mdmId
    ) {
      path += `&mdm_id=${mdmId}`;
    }

    if (
      !label &&
      !policyId &&
      !softwareId &&
      !mdmId &&
      !munkiIssueId &&
      mdmEnrollmentStatus
    ) {
      path += `&mdm_enrollment_status=${mdmEnrollmentStatus}`;
    }

    if (
      !label &&
      !policyId &&
      !softwareId &&
      !mdmId &&
      !mdmEnrollmentStatus &&
      munkiIssueId
    ) {
      path += `&munki_issue_id=${munkiIssueId}`;
    }

    if (!label && !policyId && !softwareId && !mdmId && !mdmEnrollmentStatus) {
      if (os_id) {
        path += `&os_id=${os_id}`;
      } else if (os_name && os_version) {
        path += `&os_name=${encodeURIComponent(
          os_name
        )}&os_version=${encodeURIComponent(os_version)}`;
      }
    }

    if (visibleColumns) {
      path += `&columns=${visibleColumns}`;
    }
>>>>>>> 06db6a003 (Fix munki_issue_id to munki_issue, fix pagination)

    const queryString = buildQueryStringFromParams(queryParams);
    const endpoint = endpoints.HOSTS_REPORT;
    const path = `${endpoint}?${queryString}`;

    return sendRequest("GET", path);
  },
  loadHosts: ({
    page = 0,
    perPage = 20,
    globalFilter,
    teamId,
    policyId,
    policyResponse = "passing",
    softwareId,
    mdmId,
    mdmEnrollmentStatus,
<<<<<<< HEAD
    osId,
    osName,
    osVersion,
=======
    munkiIssueId,
    os_id,
    os_name,
    os_version,
>>>>>>> 06db6a003 (Fix munki_issue_id to munki_issue, fix pagination)
    device_mapping,
    selectedLabels,
    sortBy,
  }: ILoadHostsOptions) => {
    const label = getLabel(selectedLabels);
    const sortParams = getSortParams(sortBy);

    const queryParams = {
      page,
      per_page: perPage,
      query: globalFilter,
      team_id: teamId,
      device_mapping,
      order_key: sortParams.order_key,
      order_direction: sortParams.order_direction,
      ...reconcileMutuallyExclusiveHostParams(
        label,
        policyId,
        policyResponse,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        softwareId,
        osId,
        osName,
        osVersion
      ),
      status: getStatusParam(selectedLabels),
    };

    const queryString = buildQueryStringFromParams(queryParams);

    const endpoint = getHostEndpoint(selectedLabels);
    const path = `${endpoint}?${queryString}`;
    return sendRequest("GET", path);
  },
  loadHostDetails: (hostID: number) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/${hostID}`;

    return sendRequest("GET", path);
  },
  loadHostDetailsExtension: (
    hostID: number,
    extension: ILoadHostDetailsExtension
  ) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/${hostID}/${extension}`;

    return sendRequest("GET", path);
  },
  refetch: (host: IHost) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/${host.id}/refetch`;

    return sendRequest("POST", path);
  },
  search: (searchText: string) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}?query=${searchText}`;

    return sendRequest("GET", path);
  },
  transferToTeam: (teamId: number | null, hostIds: number[]) => {
    const { HOSTS_TRANSFER } = endpoints;

    return sendRequest("POST", HOSTS_TRANSFER, {
      team_id: teamId,
      hosts: hostIds,
    });
  },

  // TODO confirm interplay with policies
  transferToTeamByFilter: (
    teamId: number | null,
    query: string,
    status: string,
    labelId: number | null
  ) => {
    const { HOSTS_TRANSFER_BY_FILTER } = endpoints;
    return sendRequest("POST", HOSTS_TRANSFER_BY_FILTER, {
      team_id: teamId,
      filters: {
        query,
        status,
        label_id: labelId,
      },
    });
  },
};
