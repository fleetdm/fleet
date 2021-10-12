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
  teamId?: number;
  policyId?: number;
  policyResponse?: string;
  selectedLabels?: string[];
}

export default {
  // hostCount.load share similar variables and parameters with hosts.loadAll
  load: (options: IHostCountLoadOptions | undefined) => {
    const { HOSTS_COUNT } = endpoints;
    const sortBy = options?.sortBy || [];
    const globalFilter = options?.globalFilter || "";
    const teamId = options?.teamId || null;
    const policyId = options?.policyId || null;
    const policyResponse = options?.policyResponse || null;
    const selectedLabels = options?.selectedLabels || [];

    let orderKeyParam = "";
    let orderDirection = "";
    if (sortBy.length !== 0) {
      const sortItem = sortBy[0];
      orderKeyParam += `order_key=${sortItem.key}`;
      orderDirection = `&order_direction=${sortItem.direction}`;
    }

    let searchQuery = "";
    if (globalFilter !== "") {
      searchQuery = `&query=${globalFilter}`;
    }

    const labelPrefix = "labels/";

    // Handle multiple filters
    const label = selectedLabels.find((f) => f.includes(labelPrefix));
    const status = selectedLabels.find((f) => !f.includes(labelPrefix));
    const isValidStatus =
      status === "new" ||
      status === "online" ||
      status === "offline" ||
      status === "mia";

    let path = `${HOSTS_COUNT}?${orderKeyParam}${orderDirection}${searchQuery}`;

    if (status && isValidStatus) {
      path += `&status=${status}`;
    }

    if (label) {
      path += `&label_id=${parseInt(label.substr(labelPrefix.length), 10)}`;
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
