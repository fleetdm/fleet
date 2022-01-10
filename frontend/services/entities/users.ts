/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";
import { INewMembersBody, IRemoveMembersBody, ITeam } from "interfaces/team";
import {
  ICreateUserFormData,
  ICreateUserFormDataNoInvite,
  IUser,
} from "interfaces/user";
import { IEnrollSecret } from "interfaces/enroll_secret";

interface ILoadAllTeamsResponse {
  teams: ITeam[];
}

interface ILoadTeamResponse {
  team: ITeam;
}

interface ITeamEnrollSecretsResponse {
  secrets: IEnrollSecret[];
}

interface IUserSearchOptions {
  page?: number;
  perPage?: number;
  globalFilter?: string;
  sortBy?: any[];
  teamId?: number;
}

interface IEditTeamFormData {
  name: string;
}

export default {
  create: (formData: ICreateUserFormData) => {
    const { USERS } = endpoints;

    return sendRequest("POST", USERS, formData).then((response) =>
      helpers.addGravatarUrlToResource(response.user)
    );
  },
  createUserWithoutInvitation: (formData: ICreateUserFormDataNoInvite) => {
    const { USERS_ADMIN } = endpoints;

    return sendRequest("POST", USERS_ADMIN, formData).then(
      (response) => helpers.addGravatarUrlToResource(response.user) // TODO: confirm
    );
  },
  deleteSessions: (user: IUser) => {
    const { USER_SESSIONS } = endpoints;
    const path = USER_SESSIONS(user.id);

    return sendRequest("DELETE", path);
  },
  destroy: (user: IUser) => {
    const { USERS } = endpoints;
    const path = `${USERS}/${user.id}`;

    return sendRequest("DELETE", path);
  },
  // TODO: forgotPassword
  // TODO: changePassword
  // TODO: confirmEmailChange
  // TODO: enable

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
  // TODO: me
  // TODO: performRequiredPasswordReset
  // TODO: requirePasswordReset
  // TODO: resetPassword
  update: (user: IUser, formData: IUpdateUserFormData) => {
    const { USERS } = endpoints;
    const path = `${USERS}/${user.id}`;

    return sendRequest("PATCH", path, formData).then((response) =>
      helpers.addGravatarUrlToResource(response.user)
    );
  },
  // TODO: updateAdmin
};
