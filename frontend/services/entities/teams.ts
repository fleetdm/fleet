/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { pick } from "lodash";

import { IEnrollSecret } from "interfaces/enroll_secret";
import {
  INewMembersBody,
  IRemoveMembersBody,
  ITeamConfig,
} from "interfaces/team";

interface ILoadTeamsParams {
  page?: number;
  perPage?: number;
  globalFilter?: string;
}

/**
 * The response body expected for the "Get team" endpoint.
 * See https://fleetdm.com/docs/using-fleet/rest-api#get-team
 */
export interface ILoadTeamResponse {
  team: ITeamConfig;
}

export interface ILoadTeamsResponse {
  teams: ITeamConfig[];
}

export interface ITeamFormData {
  name: string;
}

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
  load: (teamId: number): Promise<ILoadTeamResponse> => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}`;

    return sendRequest("GET", path);
  },
  loadAll: ({
    page = 0,
    perPage = 20,
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
  update: (
    { name, webhook_settings, integrations }: Partial<ITeamConfig>,
    teamId?: number
  ): Promise<ITeamConfig> => {
    if (typeof teamId === "undefined") {
      return Promise.reject("Invalid usage: missing team id");
    }

    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}`;
    const requestBody: Record<string, unknown> = {};
    if (name) {
      requestBody.name = name;
    }
    if (webhook_settings) {
      requestBody.webhook_settings = webhook_settings;
    }
    if (integrations) {
      const { jira, zendesk } = integrations;
      const teamIntegrationProps = [
        "enable_failing_policies",
        "group_id",
        "project_key",
        "url",
      ];
      requestBody.integrations = {
        jira: jira?.map((j) => pick(j, teamIntegrationProps)),
        zendesk: zendesk?.map((z) => pick(z, teamIntegrationProps)),
      };
    }

    return sendRequest("PATCH", path, requestBody);
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
