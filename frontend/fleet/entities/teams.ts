import endpoints from "fleet/endpoints";
import { INewMembersBody, IRemoveMembersBody, ITeam } from "interfaces/team";
import { ICreateTeamFormData } from "pages/admin/TeamManagementPage/components/CreateTeamModal/CreateTeamModal";
import { getEnrollSecrets } from "../../redux/nodes/entities/teams/actions";

interface ILoadAllTeamsResponse {
  teams: ITeam[];
}

interface ILoadTeamResponse {
  team: ITeam;
}

interface IGetTeamSecretsResponse {
  secrets: any[]; // TODO: fill this out when API is defined
}

interface ITeamSearchOptions {
  page?: number;
  perPage?: number;
  globalFilter?: string;
}

export default (client: any) => {
  return {
    create: (formData: ICreateTeamFormData) => {
      const { TEAMS } = endpoints;

      return client
        .authenticatedPost(client._endpoint(TEAMS), JSON.stringify(formData))
        .then((response: ITeam) => response);
    },

    destroy: (teamId: number) => {
      const { TEAMS } = endpoints;
      const endpoint = `${client._endpoint(TEAMS)}/${teamId}`;
      return client.authenticatedDelete(endpoint);
    },
    load: (teamId: number) => {
      const { TEAMS } = endpoints;
      const endpoint = client._endpoint(`${TEAMS}/${teamId}`);

      return client
        .authenticatedGet(endpoint)
        .then((response: ILoadTeamResponse) => response.team);
    },
    loadAll: ({
      page = 0,
      perPage = 100,
      globalFilter = "",
    }: ITeamSearchOptions = {}) => {
      const { TEAMS } = endpoints;

      // TODO: add this query param logic to client class
      const pagination = `page=${page}&per_page=${perPage}`;

      let searchQuery = "";
      if (globalFilter !== "") {
        searchQuery = `&query=${globalFilter}`;
      }

      const teamsEndpoint = `${TEAMS}?${pagination}${searchQuery}`;
      return client
        .authenticatedGet(client._endpoint(teamsEndpoint))
        .then((response: ILoadAllTeamsResponse) => {
          const { teams } = response;
          return teams;
        });
    },
    update: (teamId: number, updateParams: ITeam) => {
      const { TEAMS } = endpoints;
      const updateTeamEndpoint = `${client.baseURL}${TEAMS}/${teamId}`;

      return client
        .authenticatedPatch(updateTeamEndpoint, JSON.stringify(updateParams))
        .then((response: ITeam) => response);
    },
    addMembers: (teamId: number, newMembers: INewMembersBody) => {
      const { TEAMS_MEMBERS } = endpoints;
      return client
        .authenticatedPatch(
          client._endpoint(TEAMS_MEMBERS(teamId)),
          JSON.stringify(newMembers)
        )
        .then((response: ITeam) => response);
    },
    removeMembers: (teamId: number, removeMembers: IRemoveMembersBody) => {
      const { TEAMS_MEMBERS } = endpoints;
      return client.authenticatedDelete(
        client._endpoint(TEAMS_MEMBERS(teamId)),
        {},
        JSON.stringify(removeMembers)
      );
    },
    transferHosts: (teamId: number, hostIds: number[]) => {
      const { TEAMS_TRANSFER_HOSTS } = endpoints;
      return client
        .authenticatedPost(
          client._endpoint(TEAMS_TRANSFER_HOSTS(teamId)),
          JSON.stringify({ id: hostIds })
        )
        .then((response: ITeam) => response);
    },
    getEnrollSecrets: (teamId: number) => {
      const { TEAMS_ENROLL_SECRETS } = endpoints;
      const endpoint = client._endpoint(TEAMS_ENROLL_SECRETS(teamId));
      return client
        .authenticatedGet(endpoint)
        .then((response: IGetTeamSecretsResponse) => response.secrets);
    },
  };
};
