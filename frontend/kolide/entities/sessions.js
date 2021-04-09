import endpoints from "kolide/endpoints";
import helpers from "kolide/helpers";
import Base from "kolide/base";

export default (client) => {
  return {
    create: ({ username, password }) => {
      const { LOGIN } = endpoints;
      const endpoint = client.baseURL + LOGIN;

      return Base.post(endpoint, JSON.stringify({ username, password })).then(
        (response) => {
          const { user } = response;
          const userWithGravatarUrl = helpers.addGravatarUrlToResource(user);

          return {
            ...response,
            user: userWithGravatarUrl,
          };
        }
      );
    },
    destroy: () => {
      const { LOGOUT } = endpoints;
      const endpoint = client.baseURL + LOGOUT;

      return client.authenticatedPost(endpoint);
    },
    initializeSSO: (url) => {
      const { SSO } = endpoints;
      const endpoint = client._endpoint(SSO);
      return Base.post(endpoint, JSON.stringify({ relay_url: url }));
    },
    ssoSettings: () => {
      const { SSO } = endpoints;
      const endpoint = client._endpoint(SSO);
      return Base.get(endpoint);
    },
  };
};
