import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";

export default (client) => {
  return {
    create: (formData) => {
      const { INVITES } = endpoints;

      return client
        .authenticatedPost(client._endpoint(INVITES), JSON.stringify(formData))
        .then((response) => helpers.addGravatarUrlToResource(response.invite));
    },
    update: (invite, formData) => {
      const { INVITES } = endpoints;
      const updateInviteEndpoint = `${client.baseURL}${INVITES}/${invite.id}`;

      return client.authenticatedPatch(
        updateInviteEndpoint,
        JSON.stringify(formData)
      );
    },
    destroy: (invite) => {
      const { INVITES } = endpoints;
      const endpoint = `${client._endpoint(INVITES)}/${invite.id}`;

      return client.authenticatedDelete(endpoint);
    },
    loadAll: (page = 0, perPage = 100, globalFilter = "", sortBy = []) => {
      const { INVITES } = endpoints;

      // NOTE: this code is duplicated from /entities/users.js
      // we should pull this out into shared utility at some point.
      const pagination = `page=${page}&per_page=${perPage}`;

      let orderKeyParam = "";
      let orderDirection = "";
      if (sortBy.length !== 0) {
        const sortItem = sortBy[0];
        orderKeyParam += `&order_key=${sortItem.id}`;
        orderDirection = sortItem.desc
          ? "&order_direction=desc"
          : "&order_direction=asc";
      }

      let searchQuery = "";
      if (globalFilter !== "") {
        searchQuery = `&query=${globalFilter}`;
      }

      const inviteEndpoint = `${INVITES}?${pagination}${searchQuery}${orderKeyParam}${orderDirection}`;

      return client
        .authenticatedGet(client._endpoint(inviteEndpoint))
        .then((response) => {
          const { invites } = response;

          return invites.map((invite) => {
            return helpers.addGravatarUrlToResource(invite);
          });
        });
    },
  };
};
