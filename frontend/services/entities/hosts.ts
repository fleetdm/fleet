import { sendRequest } from "services";
import endpoints from "fleet/endpoints";
import { IHost } from "interfaces/host";

interface ISortOption {
  id: number;
  direction: string;
}

interface IHostLoadOptions {
  page: number;
  perPage: number;
  selectedLabel: string;
  globalFilter: string;
  sortBy: ISortOption[];
}

export default {
  destroy: (host: IHost) => {
    const { HOSTS } = endpoints;
    const path = `${HOSTS}/${host.id}`;

    return sendRequest("DELETE", path);
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
    const selectedLabel = options?.selectedLabel || "";
    const globalFilter = options?.globalFilter || "";
    const sortBy = options?.sortBy || [];

    // TODO: add this query param logic to client class
    const pagination = `page=${page}&per_page=${perPage}`;

    let orderKeyParam = "";
    let orderDirection = "";
    if (sortBy.length !== 0) {
      const sortItem = sortBy[0];
      orderKeyParam += `&order_key=${sortItem.id}`;
      orderDirection = `&order_direction=${sortItem.direction}`;
    }

    let searchQuery = "";
    if (globalFilter !== "") {
      searchQuery = `&query=${globalFilter}`;
    }

    let path = "";
    const labelPrefix = "labels/";
    if (selectedLabel.startsWith(labelPrefix)) {
      const lid = selectedLabel.substr(labelPrefix.length);
      path = `${LABEL_HOSTS(
        parseInt(lid, 10)
      )}?${pagination}${searchQuery}${orderKeyParam}${orderDirection}`;
    } else {
      let selectedFilter = "";
      if (
        selectedLabel === "new" ||
        selectedLabel === "online" ||
        selectedLabel === "offline" ||
        selectedLabel === "mia"
      ) {
        selectedFilter = `&status=${selectedLabel}`;
      }
      path = `${HOSTS}?${pagination}${selectedFilter}${searchQuery}${orderKeyParam}${orderDirection}`;
    }

    return sendRequest("GET", path);
  },
  transferToTeam: (teamId: number | null, hostIds: number[]) => {
    const { HOSTS_TRANSFER } = endpoints;

    return sendRequest("POST", HOSTS_TRANSFER, {
      team_id: teamId,
      hosts: hostIds,
    });
  },
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
