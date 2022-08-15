/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { IHost } from "interfaces/host";
import { buildQueryStringFromParams } from "utilities/url";

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
  operatingSystemId?: number;
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
  operatingSystemId?: number;
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
  if (selectedLabels === undefined) return endpoints.HOSTS;

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
  softwareId?: number
) => {
  return label === undefined && policyId === undefined ? softwareId : undefined;
};

const getOperatingSystemParam = (
  label?: string,
  policyId?: number,
  softwareId?: number,
  operatingSystemId?: number
) => {
  return label === undefined &&
    policyId === undefined &&
    softwareId === undefined
    ? operatingSystemId
    : undefined;
};

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
    const { HOSTS_REPORT } = endpoints;
    const sortBy = options.sortBy;
    const selectedLabels = options?.selectedLabels || [];
    const globalFilter = options?.globalFilter || "";
    const teamId = options?.teamId || null;
    const policyId = options?.policyId || null;
    const policyResponse = options?.policyResponse || "passing";
    const softwareId = options?.softwareId || null;
    const visibleColumns = options?.visibleColumns || null;

    if (!sortBy.length) {
      throw Error("sortBy is a required field.");
    }

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

    // Label OR policy_id OR software_id are valid filters.
    if (label) {
      const lid = label.substr(labelPrefix.length);
      path += `&label_id=${parseInt(lid, 10)}`;
    }

    if (!label && policyId) {
      path += `&policy_id=${policyId}`;
      path += `&policy_response=${policyResponse}`;
    }

    if (!label && !policyId && softwareId) {
      path += `&software_id=${softwareId}`;
    }

    if (visibleColumns) {
      path += `&columns=${visibleColumns}`;
    }

    path += "&format=csv";

    return sendRequest("GET", path);
  },
  loadHosts: ({
    page = 0,
    perPage = 100,
    globalFilter,
    teamId,
    policyId,
    policyResponse = "passing",
    softwareId,
    operatingSystemId,
    device_mapping,
    selectedLabels,
    sortBy,
  }: ILoadHostsOptions) => {
    const label = getLabel(selectedLabels);
    const sortParams = getSortParams(sortBy);
    const policyParams = getPolicyParams(label, policyId, policyResponse);

    const queryParams = {
      page,
      per_page: perPage,
      query: globalFilter,
      team_id: teamId,
      device_mapping,
      order_key: sortParams.order_key,
      order_direction: sortParams.order_direction,
      policy_id: policyParams.policy_id,
      policy_response: policyParams.policy_response,
      software_id: getSoftwareParam(label, policyId, softwareId),
      operating_system_id: getOperatingSystemParam(
        label,
        policyId,
        softwareId,
        operatingSystemId
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
