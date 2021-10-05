/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { IHost } from "interfaces/host";

export interface ISortOption {
  key: string;
  direction: string;
}

export interface IHostCountLoadOptions {
  page?: number;
  perPage?: number;
  sortBy?: ISortOption[];
  status?: string;
  globalFilter?: string;
  additionalInfoFilters?: string;
  teamId?: number;
  policyId?: number;
  policyResponse?: string;
  selectedLabels?: string[];
}

export default {
  // hostCount.load share similar variables and parameters with hosts.loadAll
  load: (options: IHostCountLoadOptions | undefined) => {
    const { HOSTS_COUNT, LABEL_HOSTS_COUNT } = endpoints;
    const page = options?.page || 0;
    const perPage = options?.perPage || 100;
    const sortBy = options?.sortBy || [];
    const globalFilter = options?.globalFilter || "";
    const additionalInfoFilters = options?.additionalInfoFilters;
    const teamId = options?.teamId || null;
    const policyId = options?.policyId || null;
    const policyResponse = options?.policyResponse || null;
    const selectedLabels = options?.selectedLabels || [];

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
      path = `${LABEL_HOSTS_COUNT(
        parseInt(lid, 10)
      )}?${pagination}${searchQuery}${orderKeyParam}${orderDirection}`;

      // connect status if applicable
      if (status && isValidStatus) {
        path += `&status=${status}`;
      }
    } else if (status && isValidStatus) {
      path = `${HOSTS_COUNT}?${pagination}&status=${status}${searchQuery}${orderKeyParam}${orderDirection}`;
    } else {
      path = `${HOSTS_COUNT}?${pagination}${searchQuery}${orderKeyParam}${orderDirection}`;
    }

    if (teamId) {
      path += `&team_id=${teamId}`;
    }

    if (!label && policyId) {
      path += `&policy_id=${policyId}`;
      path += `&policy_response=${policyResponse || "passing"}`; // TODO confirm whether there should be a default if there is an id but no response specified
    }

    return sendRequest("GET", path);
  },
};
