/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";
import {
  ICreateUserFormData,
  IUpdateUserFormData,
  IUser,
} from "interfaces/user";
import { ITeamSummary } from "interfaces/team";

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

interface IRequirePasswordReset {
  require: boolean;
}

export interface IGetMeResponse {
  user: IUser;
  available_teams: ITeamSummary[];
}

export default {
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
  forgotPassword: ({ email }: IForgotPassword) => {
    const { FORGOT_PASSWORD } = endpoints;

    return sendRequest("POST", FORGOT_PASSWORD, { email });
  },
  // TODO: changePassword (UserSettingsPage.jsx refactor)
  loadAll: ({
    page = 0,
    perPage = 100,
    globalFilter = "",
    sortBy = [],
    teamId,
  }: IUserSearchOptions = {}) => {
    const { USERS } = endpoints;

    // TODO: add this query param logic to client class
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

    let teamQuery = "";
    if (teamId !== undefined) {
      teamQuery = `&team_id=${teamId}`;
    }

    const path = `${USERS}?${pagination}${searchQuery}${orderKeyParam}${orderDirection}${teamQuery}`;

    return sendRequest("GET", path).then((response) => {
      const { users } = response;

      return users.map((u: IUser) => helpers.addGravatarUrlToResource(u));
    });
  },
  me: (): Promise<IGetMeResponse> => {
    const { ME } = endpoints;

    return sendRequest("GET", ME).then(({ user, available_teams }) => {
      return {
        user: helpers.addGravatarUrlToResource(user),
        available_teams,
      };
    });
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
  update: (userId: number, formData: IUpdateUserFormData) => {
    const { USERS } = endpoints;
    const path = `${USERS}/${userId}`;

    return sendRequest("PATCH", path, formData).then((response) =>
      helpers.addGravatarUrlToResource(response.user)
    );
  },
};
