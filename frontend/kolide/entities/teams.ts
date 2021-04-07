import endpoints from 'kolide/endpoints';

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Base from 'kolide/base';

import { ITeam } from 'interfaces/team';

interface ITeamsResponse {
  teams: ITeam[];
}

export default (client: any) => {
  return {
    create: () => {
      return {};
    },

    loadAll: (page = 0, perPage = 100, globalFilter = '') => {
      const { TEAMS } = endpoints;

      // TODO: add this query param logic to client class
      const pagination = `page=${page}&per_page=${perPage}`;

      let searchQuery = '';
      if (globalFilter !== '') {
        searchQuery = `&query=${globalFilter}`;
      }

      const teamsEndpoint = `${TEAMS}?${pagination}${searchQuery}`;
      return client.authenticatedGet(client._endpoint(teamsEndpoint))
        .then((response: ITeamsResponse) => {
          const { teams } = response;
          return teams;
        });
    },

    update: () => {
      return {};
    },
  };
};
