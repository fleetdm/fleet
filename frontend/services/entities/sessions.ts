import sendRequest from "services";
import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";

interface ICreateSessionProps {
  email: string;
  password: string;
}

export default {
  create: ({ email, password }: ICreateSessionProps) => {
    const { LOGIN } = endpoints;

    return sendRequest("POST", LOGIN, JSON.stringify({ email, password })).then((response) => {
      const { user, available_teams } = response;
      const userWithGravatarUrl = helpers.addGravatarUrlToResource(user);

      return {
        ...response,
        user: userWithGravatarUrl,
        available_teams,
      };
    });
  },
  destroy: () => {
    const { LOGOUT } = endpoints;
    return sendRequest("POST", LOGOUT);
  },
  initializeSSO: (relay_url: string) => {
    const { SSO } = endpoints;
    return sendRequest("POST", SSO, JSON.stringify({ relay_url }));
  },
  ssoSettings: () => {
    const { SSO } = endpoints;
    return sendRequest("GET", SSO);
  },
};