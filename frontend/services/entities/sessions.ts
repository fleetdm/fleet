import { ISSOSettings } from "interfaces/ssoSettings";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import helpers from "utilities/helpers";

interface ILoginProps {
  email: string;
  password: string;
}

interface ICreateSessionProps {
  token: string;
}

export interface ISSOSettingsResponse {
  settings: ISSOSettings;
}

export default {
  login: ({ email, password }: ILoginProps) => {
    const { LOGIN } = endpoints;

    return sendRequest(
      "POST",
      LOGIN,
      {
        email,
        password,
        supports_email_verification: true,
      },
      "json",
      undefined,
      undefined,
      true // returns raw data which includes the status code alongside data
    ).then((rawResponse) => {
      if (rawResponse.status === 202) {
        // MFA; treat as an error and let the caller handle it
        throw rawResponse;
      }
      const response = rawResponse.data;
      const { user, available_teams } = response;
      const userWithGravatarUrl = helpers.addGravatarUrlToResource(user);

      return {
        ...response,
        user: userWithGravatarUrl,
        available_teams,
      };
    });
  },
  finishMFA: ({ token }: ICreateSessionProps) => {
    const { CREATE_SESSION } = endpoints;

    return sendRequest("POST", CREATE_SESSION, {
      token,
    }).then((response) => {
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
    return sendRequest("POST", SSO, { relay_url });
  },
  ssoSettings: (): Promise<ISSOSettingsResponse> => {
    const { SSO } = endpoints;
    return sendRequest("GET", SSO);
  },
};
