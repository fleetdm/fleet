/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import helpers from "utilities/helpers";
import { buildQueryStringFromParams } from "utilities/url";

import {
  IInvite,
  ICreateInviteFormData,
  IEditInviteFormData,
} from "interfaces/invite";

export interface ISortOption {
  id: number;
  desc: boolean;
}

interface IInviteSearchOptions {
  page?: number;
  perPage?: number;
  globalFilter?: string;
  sortBy?: ISortOption[];
}

export interface IValidateInviteResp {
  invite: IInvite;
}

export default {
  create: (formData: ICreateInviteFormData) => {
    const { INVITES } = endpoints;

    return sendRequest("POST", INVITES, formData).then((response) =>
      helpers.addGravatarUrlToResource(response.invite)
    );
  },
  update: (inviteId: number, formData: IEditInviteFormData) => {
    const { INVITES } = endpoints;
    const path = `${INVITES}/${inviteId}`;

    return sendRequest("PATCH", path, formData);
  },
  destroy: (inviteId: number) => {
    const { INVITES } = endpoints;
    const path = `${INVITES}/${inviteId}`;

    return sendRequest("DELETE", path);
  },
  verify: (token: string): Promise<IValidateInviteResp> => {
    return sendRequest("GET", endpoints.INVITE_VERIFY(token));
  },
  loadAll: ({ globalFilter = "" }: IInviteSearchOptions) => {
    const queryParams = {
      query: globalFilter,
    };

    const queryString = buildQueryStringFromParams(queryParams);
    const endpoint = endpoints.INVITES;
    const path = `${endpoint}?${queryString}`;

    return sendRequest("GET", path).then((response) => {
      const { invites } = response;

      return invites.map((invite: IInvite) => {
        return helpers.addGravatarUrlToResource(invite);
      });
    });
  },
};
