import endpoints from 'kolide/endpoints';
import { ITeam } from 'interfaces/team';
import { ICreateTeamFormData } from 'pages/admin/TeamManagementPage/components/CreateTeamModal/CreateTeamModal';

interface ITeamsResponse {
  teams: ITeam[];
}

export default (client: any) => {
  return {
    create: (formData: ICreateTeamFormData) => {
      const { TEAMS } = endpoints;

      return client.authenticatedPost(client._endpoint(TEAMS), JSON.stringify(formData))
        .then((response: ITeam) => response);
    },

    destroy: (teamId: number) => {
      const { TEAMS } = endpoints;
      const endpoint = `${client._endpoint(TEAMS)}/${teamId}`;
      return client.authenticatedDelete(endpoint);
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
    update: (teamId: number, updateParams: ITeam) => {
      const { TEAMS } = endpoints;
      const updateQueryEndpoint = `${client.baseURL}${TEAMS}/${teamId}`;

      return client.authenticatedPatch(updateQueryEndpoint, JSON.stringify(updateParams))
        .then((response: ITeam) => response);
    },
  };
};
