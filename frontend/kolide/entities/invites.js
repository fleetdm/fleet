import endpoints from "kolide/endpoints";
import helpers from "kolide/helpers";

export default (client) => {
  return {
    create: (formData) => {
      const { INVITES } = endpoints;

      return client
        .authenticatedPost(client._endpoint(INVITES), JSON.stringify(formData))
        .then((response) => helpers.addGravatarUrlToResource(response.invite));
    },
    destroy: (invite) => {
      const { INVITES } = endpoints;
      const endpoint = `${client._endpoint(INVITES)}/${invite.id}`;

      return client.authenticatedDelete(endpoint);
    },
    loadAll: () => {
      const { INVITES } = endpoints;

      return client
        .authenticatedGet(client._endpoint(INVITES))
        .then((response) => {
          const { invites } = response;

          return invites.map((invite) => {
            return helpers.addGravatarUrlToResource(invite);
          });
        });
    },
  };
};
