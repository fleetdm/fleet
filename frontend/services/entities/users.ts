/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";
import {
  ICreateUserFormDataNoInvite,
  IUpdateUserFormData,
  IDeleteSessionsUser,
  IDestroyUser,
  IUser,
} from "interfaces/user";
import { IInvite } from "interfaces/invite";

interface IUserSearchOptions {
  page?: number;
  perPage?: number;
  globalFilter?: string;
  sortBy?: any[];
  teamId?: number;
}

interface IForgotPassword {
  email: string;
}
interface IEnable {
  enabled: boolean;
}

interface IUpdateAdmin {
  admin: boolean;
}

interface IRequirePasswordReset {
  require: boolean;
}

// TODO
// interface IResetPassword {
// }

export default {
  createUserWithoutInvitation: (formData: ICreateUserFormDataNoInvite) => {
    const { USERS_ADMIN } = endpoints;

    return sendRequest("POST", USERS_ADMIN, formData).then(
      (response) => helpers.addGravatarUrlToResource(response.user) // TODO: confirm
    );
  },
  deleteSessions: (user: IDeleteSessionsUser) => {
    const { USER_SESSIONS } = endpoints;
    const path = USER_SESSIONS(user.id);

    return sendRequest("DELETE", path);
  },
  destroy: (user: IDestroyUser) => {
    const { USERS } = endpoints;
    const path = `${USERS}/${user.id}`;

    return sendRequest("DELETE", path);
  },
  forgotPassword: ({ email }: IForgotPassword) => {
    const { FORGOT_PASSWORD } = endpoints;

    return sendRequest("POST", FORGOT_PASSWORD, { email });
  },
  // TODO: changePassword (UserSettingsPage.jsx refactor)
  // TODO: confirmEmailChange
  enable: (user: IUser, { enabled }: IEnable) => {
    const { ENABLE_USER } = endpoints;
    const path = ENABLE_USER(user.id);

    return sendRequest("POST", path, { enabled }).then(
      (response) => helpers.addGravatarUrlToResource(response.user) // TODO: confirm
    );
  },
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
  me: () => {
    const { ME } = endpoints;

    return sendRequest("GET", ME).then((response) =>
      helpers.addGravatarUrlToResource(response.user)
    );
  },
  // TODO: performRequiredPasswordReset
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
  update: (
    user: IUser | IInvite | undefined,
    formData: IUpdateUserFormData
  ) => {
    const { USERS } = endpoints;
    const path = `${USERS}/${user?.id}`;

    return sendRequest("PATCH", path, formData).then((response) =>
      helpers.addGravatarUrlToResource(response.user)
    );
  },
  updateAdmin: (user: IUser, { admin }: IUpdateAdmin) => {
    const { UPDATE_USER_ADMIN } = endpoints;
    const path = UPDATE_USER_ADMIN(user.id);

    return sendRequest("POST", path, { admin }).then((response) =>
      helpers.addGravatarUrlToResource(response.user)
    );
  },
};
