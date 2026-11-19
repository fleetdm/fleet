/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */

import {
  IInvite,
  ICreateInviteFormData,
  IEditInviteFormData,
} from "interfaces/invite";
import sendRequest from "services";
import { fetchAllPages } from "services/entities/common";
import { toFleetRoles } from "services/entities/users";
import endpoints from "utilities/endpoints";
import helpers from "utilities/helpers";
import { buildQueryStringFromParams } from "utilities/url";

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

export interface IValidateInviteResponse {
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
    const body = formData.fleets
      ? { ...formData, fleets: toFleetRoles(formData.fleets) }
      : formData;

    return sendRequest("PATCH", path, body);
  },
  destroy: (inviteId: number) => {
    const { INVITES } = endpoints;
    const path = `${INVITES}/${inviteId}`;

    return sendRequest("DELETE", path);
  },
  verify: (token: string): Promise<IValidateInviteResponse> => {
    return sendRequest("GET", endpoints.INVITE_VERIFY(token));
  },
  /** Fetches every invite, page by page — duplicate-invite detection needs
   * the complete list. */
  loadAll: async ({ globalFilter = "" }: IInviteSearchOptions) => {
    const invites = await fetchAllPages<IInvite>(async (page, perPage) => {
      const queryString = buildQueryStringFromParams({
        query: globalFilter,
        page,
        per_page: perPage,
      });
      const response = await sendRequest(
        "GET",
        `${endpoints.INVITES}?${queryString}`
      );
      return response.invites;
    });
    return invites.map((invite: IInvite) =>
      helpers.addGravatarUrlToResource(invite)
    );
  },
};
