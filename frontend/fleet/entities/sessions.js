import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";
import Base from "fleet/base";

export default (client) => {
  return {
    create: ({ email, password }) => {
      const { LOGIN } = endpoints;
      const endpoint = client.baseURL + LOGIN;

      return Base.post(endpoint, JSON.stringify({ email, password })).then(
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
