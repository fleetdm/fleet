/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import { IEnrollSecret } from "interfaces/enroll_secret";
import { INewMembersBody, IRemoveMembersBody, ITeam } from "interfaces/team";
import endpoints from "utilities/endpoints";
import { IWebhook } from "interfaces/webhook";

interface ILoadTeamsParams {
  page?: number;
  perPage?: number;
  globalFilter?: string;
}

export interface ILoadTeamsResponse {
  teams: ITeam[];
}

export interface ITeamFormData {
  name: string;
}

interface ITeamWebhooks {
  webhook_settings: {
    [key: string]: IWebhook;
  };
}

type ITeamUpdateData = ITeamFormData | ITeamWebhooks;

export default {
  create: (formData: ITeamFormData) => {
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
  }: ILoadTeamsParams = {}): Promise<ILoadTeamsResponse> => {
    const { TEAMS } = endpoints;

    // TODO: add this query param logic to client class
    const pagination = `page=${page}&per_page=${perPage}`;

    let searchQuery = "";
    if (globalFilter !== "") {
      searchQuery = `&query=${globalFilter}`;
    }

    const path = `${TEAMS}?${pagination}${searchQuery}`;

    return sendRequest("GET", path);
  },
  update: (updateParams: ITeamUpdateData, teamId?: number) => {
    // we are grouping this update with the config api update function
    // on the ManagePoliciesPage to streamline updating the
    // webhook settings globally or for a team - see ManagePoliciesPage line 208
    if (typeof teamId === "undefined") {
      return Promise.reject("Invalid usage: missing team id");
    }

    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}`;

    return sendRequest("PATCH", path, updateParams);
  },
  addMembers: (teamId: number, newMembers: INewMembersBody) => {
    const { TEAMS_MEMBERS } = endpoints;
    const path = TEAMS_MEMBERS(teamId);

    return sendRequest("PATCH", path, newMembers);
  },
  removeMembers: (teamId: number, removeMembers: IRemoveMembersBody) => {
    const { TEAMS_MEMBERS } = endpoints;
    const path = TEAMS_MEMBERS(teamId);

    return sendRequest("DELETE", path, removeMembers);
  },
  transferHosts: (teamId: number, hostIds: number[]) => {
    const { TEAMS_TRANSFER_HOSTS } = endpoints;
    const path = TEAMS_TRANSFER_HOSTS(teamId);

    return sendRequest("POST", path, { id: hostIds });
  },
  getEnrollSecrets: (teamId: number) => {
    const { TEAMS_ENROLL_SECRETS } = endpoints;
    const path = TEAMS_ENROLL_SECRETS(teamId);

    return sendRequest("GET", path);
  },
  modifyEnrollSecrets: (teamId: number, secrets: IEnrollSecret[]) => {
    const { TEAMS_ENROLL_SECRETS } = endpoints;
    const path = TEAMS_ENROLL_SECRETS(teamId);

    return sendRequest("PATCH", path, { secrets });
  },
};
