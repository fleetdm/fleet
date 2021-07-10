import endpoints from "fleet/endpoints";
import { IHost } from "interfaces/host";

interface ISortOption {
  id: number;
  direction: string;
}

export default (client: any) => {
  return {
    destroy: (host: IHost) => {
      const { HOSTS } = endpoints;
      const endpoint = client._endpoint(`${HOSTS}/${host.id}`);

      return client.authenticatedDelete(endpoint);
    },
    refetch: (host: IHost) => {
      const { HOSTS } = endpoints;
      const endpoint = client._endpoint(`${HOSTS}/${host.id}/refetch`);

      return client
        .authenticatedPost(endpoint)
        .then((response: any) => response.host);
    },
    load: (hostID: number) => {
      const { HOSTS } = endpoints;
      const endpoint = client._endpoint(`${HOSTS}/${hostID}`);

      return client
        .authenticatedGet(endpoint)
        .then((response: any) => response.host);
    },
    loadAll: (
      page = 0,
      perPage = 100,
      selected = "",
      globalFilter = "",
      sortBy: ISortOption[] = []
    ) => {
      const { HOSTS, LABEL_HOSTS } = endpoints;

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

      let endpoint = "";
      const labelPrefix = "labels/";
      if (selected.startsWith(labelPrefix)) {
        const lid = selected.substr(labelPrefix.length);
        endpoint = `${LABEL_HOSTS(
          parseInt(lid, 10)
        )}?${pagination}${searchQuery}${orderKeyParam}${orderDirection}`;
      } else {
        let selectedFilter = "";
        if (
          selected === "new" ||
          selected === "online" ||
          selected === "offline" ||
          selected === "mia"
        ) {
          selectedFilter = `&status=${selected}`;
        }
        endpoint = `${HOSTS}?${pagination}${selectedFilter}${searchQuery}${orderKeyParam}${orderDirection}`;
      }

      return client
        .authenticatedGet(client._endpoint(endpoint))
        .then((response: any) => {
          return response.hosts;
        });
    },
    transferToTeam: (teamId: number | null, hostIds: number[]) => {
      const { HOSTS_TRANSFER } = endpoints;
      const endpoint = client._endpoint(HOSTS_TRANSFER);
      return client.authenticatedPost(
        endpoint,
        JSON.stringify({
          team_id: teamId,
          hosts: hostIds,
        })
      );
    },
    transferToTeamByFilter: (
      teamId: number | null,
      query: string,
      status: string,
      labelId: number | null
    ) => {
      const { HOSTS_TRANSFER_BY_FILTER } = endpoints;
      const endpoint = client._endpoint(HOSTS_TRANSFER_BY_FILTER);
      return client.authenticatedPost(
        endpoint,
        JSON.stringify({
          team_id: teamId,
          filters: {
            query,
            status,
            label_id: labelId,
          },
        })
      );
    },
  };
};
