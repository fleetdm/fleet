/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";
import {
  IInvite,
  ICreateInviteFormData,
  IEditInviteFormData,
} from "interfaces/invite";

interface IInviteSearchOptions {
  page?: number;
  perPage?: number;
  globalFilter?: string;
  sortBy?: any[];
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
  loadAll: ({
    page = 0,
    perPage = 100,
    globalFilter = "",
    sortBy = [],
  }: IInviteSearchOptions) => {
    const { INVITES } = endpoints;

    // NOTE: this code is duplicated from /entities/users.js
    // we should pull this out into shared utility at some point.
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

    const path = `${INVITES}?${pagination}${searchQuery}${orderKeyParam}${orderDirection}`;

    return sendRequest("GET", path).then((response) => {
      const { invites } = response;

      return invites.map((invite: IInvite) => {
        return helpers.addGravatarUrlToResource(invite);
      });
    });
  },
};
