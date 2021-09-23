import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { INewMembersBody, IRemoveMembersBody, ITeam } from "interfaces/team";
import { ICreateTeamFormData } from "pages/admin/TeamManagementPage/components/CreateTeamModal/CreateTeamModal";

interface ITeamSearchOptions {
  page?: number;
  perPage?: number;
  globalFilter?: string;
}

export default {
  create: (formData: ICreateTeamFormData) => {
    const { TEAMS } = endpoints;

    return sendRequest("POST", TEAMS, formData);
  },
  destroy: (teamId: number) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}`;

    return sendRequest("DELETE", path);
  },
  load: (teamId: number) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}`;

    return sendRequest("GET", path);
  },
  loadAll: ({
    page = 0,
    perPage = 100,
    globalFilter = "",
  }: ITeamSearchOptions = {}) => {
    const { TEAMS } = endpoints;

    // TODO: add this query param logic to client class - is this outdated todo 9/17 RP
    const pagination = `page=${page}&per_page=${perPage}`;

    let searchQuery = "";
    if (globalFilter !== "") {
      searchQuery = `&query=${globalFilter}`;
    }

    const path = `${TEAMS}?${pagination}${searchQuery}`;

    return sendRequest("GET", path);
  },
  update: (teamId: number, updatedAttrs: ITeam) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}`;

    return sendRequest("PATCH", path, updatedAttrs);
  },
  addMembers: (teamId: number, newMembers: INewMembersBody) => {
    const { TEAMS_MEMBERS } = endpoints;
    return sendRequest("PATCH", TEAMS_MEMBERS(teamId), newMembers);
  },
  removeMembers: (teamId: number, removeMembers: IRemoveMembersBody) => {
    const { TEAMS_MEMBERS } = endpoints;

    return sendRequest("DELETE", TEAMS_MEMBERS(teamId), removeMembers);
  },
  transferHosts: (teamId: number, hostIds: number[]) => {
    const { TEAMS_TRANSFER_HOSTS } = endpoints;

    return sendRequest("POST", TEAMS_TRANSFER_HOSTS(teamId), { id: hostIds });
  },
  getEnrollSecrets: (teamId: number) => {
    const { TEAMS_ENROLL_SECRETS } = endpoints;
    const path = TEAMS_ENROLL_SECRETS(teamId);

    return sendRequest("GET", path, teamId);
  },
};