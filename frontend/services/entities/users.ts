/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import helpers from "utilities/helpers";
import { buildQueryStringFromParams } from "utilities/url";

import {
  ICreateUserFormData,
  IUpdateUserFormData,
  IUser,
  ICreateUserWithInvitationFormData,
} from "interfaces/user";
import { ITeamSummary } from "interfaces/team";
import { IUserSettings } from "interfaces/config";

export interface ISortOption {
  id: number;
  desc: boolean;
}

interface IUserSearchOptions {
  page?: number;
  perPage?: number;
  globalFilter?: string;
  sortBy?: ISortOption[];
  teamId?: number;
}

interface IForgotPassword {
  email: string;
}

interface IUpdatePassword {
  new_password: string;
  old_password: string;
}

interface IRequirePasswordReset {
  require: boolean;
}

export interface IGetMeResponse {
  user: IUser;
  available_teams: ITeamSummary[];
  settings: IUserSettings;
}

export default {
  changePassword: (passwordParams: IUpdatePassword) => {
    const { CHANGE_PASSWORD } = endpoints;

    return sendRequest("POST", CHANGE_PASSWORD, passwordParams);
  },
  confirmEmailChange: (currentUser: IUser, token: string) => {
    const { CONFIRM_EMAIL_CHANGE } = endpoints;

    return sendRequest("GET", CONFIRM_EMAIL_CHANGE(token)).then((response) => {
      return { ...currentUser, email: response.new_email };
    });
  },
  create: (formData: ICreateUserWithInvitationFormData) => {
    const { USERS } = endpoints;

    return sendRequest("POST", USERS, formData).then((response) =>
      helpers.addGravatarUrlToResource(response.user)
    );
  },
  createUserWithoutInvitation: (formData: ICreateUserFormData) => {
    const { USERS_ADMIN } = endpoints;

    return sendRequest("POST", USERS_ADMIN, formData).then((response) =>
      helpers.addGravatarUrlToResource(response.user)
    );
  },
  deleteSessions: (userId: number) => {
    const { USER_SESSIONS } = endpoints;
    const path = USER_SESSIONS(userId);

    return sendRequest("DELETE", path);
  },
  destroy: (userId: number) => {
    const { USERS } = endpoints;
    const path = `${USERS}/${userId}`;

    return sendRequest("DELETE", path);
  },
  enable: (user: IUser, enabled: boolean) => {
    const { ENABLE_USER } = endpoints;

    return sendRequest("POST", ENABLE_USER(user.id), {
      enabled,
    }).then((response) => helpers.addGravatarUrlToResource(response.user));
  },
  forgotPassword: ({ email }: IForgotPassword) => {
    const { FORGOT_PASSWORD } = endpoints;

    return sendRequest("POST", FORGOT_PASSWORD, { email });
  },
  loadAll: ({ globalFilter = "", teamId }: IUserSearchOptions = {}) => {
    const queryParams = {
      query: globalFilter,
      team_id: teamId,
    };

    const queryString = buildQueryStringFromParams(queryParams);
    const endpoint = endpoints.USERS;
    const path = `${endpoint}?${queryString}`;

    return sendRequest("GET", path).then((response) => {
      const { users } = response;

      return users.map((u: IUser) => helpers.addGravatarUrlToResource(u));
    });
  },
  me: (): Promise<IGetMeResponse> => {
    // include the user's settings when calling from the UI
    const path = `${endpoints.ME}?include_settings=true`;
    return sendRequest("GET", path).then(
      ({ user, available_teams, settings }) => {
        return {
          user: helpers.addGravatarUrlToResource(user),
          available_teams,
          settings,
        };
      }
    );
  },
  performRequiredPasswordReset: (new_password: string) => {
    const { PERFORM_REQUIRED_PASSWORD_RESET } = endpoints;

    return sendRequest("POST", PERFORM_REQUIRED_PASSWORD_RESET, {
      new_password,
    }).then((response) => helpers.addGravatarUrlToResource(response.user));
  },
  requirePasswordReset: (
    userId: number,
    { require }: IRequirePasswordReset
  ) => {
    const { USERS } = endpoints;
    const path = `${USERS}/${userId}/require_password_reset`;

    return sendRequest("POST", path, { require }).then((response) =>
      helpers.addGravatarUrlToResource(response.user)
    );
  },
  resetPassword: (formData: any) => {
    const { RESET_PASSWORD } = endpoints;

    return sendRequest("POST", RESET_PASSWORD, formData);
  },
  setup: (formData: any) => {
    const { SETUP } = endpoints;
    const setupData = helpers.setupData(formData);

    return sendRequest("POST", SETUP, setupData);
  },
  update: (userId: number, formData: IUpdateUserFormData) => {
    const { USERS } = endpoints;
    const path = `${USERS}/${userId}`;

    return sendRequest("PATCH", path, formData).then((response) =>
      helpers.addGravatarUrlToResource(response.user)
    );
  },
  updateAdmin: (user: IUser, admin: boolean) => {
    const { UPDATE_USER_ADMIN } = endpoints;

    return sendRequest(
      "POST",
      UPDATE_USER_ADMIN(user.id),
      admin
    ).then((response) => helpers.addGravatarUrlToResource(response.user));
  },
};
