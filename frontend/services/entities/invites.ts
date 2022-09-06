/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import helpers from "utilities/helpers";
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
  loadAll: ({ globalFilter = "", sortBy = [] }: IInviteSearchOptions) => {
    const { INVITES } = endpoints;

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
      searchQuery = `query=${globalFilter}`;
    }

    const path = `${INVITES}?${searchQuery}${orderKeyParam}${orderDirection}`;

    return sendRequest("GET", path).then((response) => {
      const { invites } = response;

      return invites.map((invite: IInvite) => {
        return helpers.addGravatarUrlToResource(invite);
      });
    });
  },
};
