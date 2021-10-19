/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { IHost } from "interfaces/host";

export interface ISortOption {
  key: string;
  direction: string;
}

export interface IHostLoadOptions {
  page?: number;
  perPage?: number;
  selectedLabels?: string[];
  globalFilter?: string;
  sortBy?: ISortOption[];
  teamId?: number;
  policyId?: number;
  policyResponse?: string;
  softwareId?: number;
}

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
  refetch: (host: IHost) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/${host.id}/refetch`;

    return sendRequest("POST", path);
  },
  load: (hostID: number) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/${hostID}`;

    return sendRequest("GET", path);
  },
  loadAll: (options: IHostLoadOptions | undefined) => {
    const { HOSTS, LABEL_HOSTS } = endpoints;
    const page = options?.page || 0;
    const perPage = options?.perPage || 100;
    const selectedLabels = options?.selectedLabels || [];
    const globalFilter = options?.globalFilter || "";
    const sortBy = options?.sortBy || [];
    const teamId = options?.teamId || null;
    const policyId = options?.policyId || null;
    const policyResponse = options?.policyResponse || null;
    const softwareId = options?.softwareId || null;

    // TODO: add this query param logic to client class
    const pagination = `page=${page}&per_page=${perPage}`;

    let orderKeyParam = "";
    let orderDirection = "";
    if (sortBy.length !== 0) {
      const sortItem = sortBy[0];
      orderKeyParam += `&order_key=${sortItem.key}`;
      orderDirection = `&order_direction=${sortItem.direction}`;
    }

    let searchQuery = "";
    if (globalFilter !== "") {
      searchQuery = `&query=${globalFilter}`;
    }

    let path = "";
    const labelPrefix = "labels/";

    // Handle multiple filters
    const label = selectedLabels.find((f) => f.includes(labelPrefix));
    const status = selectedLabels.find((f) => !f.includes(labelPrefix));
    const isValidStatus =
      status === "new" ||
      status === "online" ||
      status === "offline" ||
      status === "mia";

    if (label) {
      const lid = label.substr(labelPrefix.length);
      path = `${LABEL_HOSTS(
        parseInt(lid, 10)
      )}?${pagination}${searchQuery}${orderKeyParam}${orderDirection}`;

      // connect status if applicable
      if (status && isValidStatus) {
        path += `&status=${status}`;
      }
    } else if (status && isValidStatus) {
      path = `${HOSTS}?${pagination}&status=${status}${searchQuery}${orderKeyParam}${orderDirection}`;
    } else {
      path = `${HOSTS}?${pagination}${searchQuery}${orderKeyParam}${orderDirection}`;
    }

    if (teamId) {
      path += `&team_id=${teamId}`;
    }

    if (!label && policyId) {
      path += `&policy_id=${policyId}`;
      path += `&policy_response=${policyResponse || "passing"}`; // TODO: confirm whether there should be a default if there is an id but no response sepcified
    }
    // TODO: consider how to check for mutually exclusive scenarios with label, policy and software
    if (!label && !policyId && softwareId) {
      path += `&software_id=${softwareId}`;
    }

    return sendRequest("GET", path);
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
