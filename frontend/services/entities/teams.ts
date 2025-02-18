/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { pick } from "lodash";

import { buildQueryStringFromParams } from "utilities/url";
import { IEnrollSecret } from "interfaces/enroll_secret";
import { ITeamIntegrations } from "interfaces/integration";
import {
  API_NO_TEAM_ID,
  INewTeamUsersBody,
  IRemoveTeamUserBody,
  ITeamConfig,
  ITeamWebhookSettings,
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

export interface IUpdateTeamFormData {
  name: string;
  webhook_settings: Partial<ITeamWebhookSettings>;
  integrations: ITeamIntegrations;
  mdm: {
    macos_updates?: {
      minimum_version: string;
      deadline: string;
    };
    ios_updates?: {
      minimum_version: string;
      deadline: string;
    };
    ipados_updates?: {
      minimum_version: string;
      deadline: string;
    };
    windows_updates?: {
      deadline_days: number;
      grace_period_days: number;
    };
  };
  host_expiry_settings: {
    host_expiry_enabled: boolean;
    host_expiry_window: number; // days
  };
}

export default {
  create: (formData: ITeamFormData) => {
    const { TEAMS } = endpoints;

    if (formData.name) {
      formData.name = formData.name.trim();
    }

    return sendRequest("POST", TEAMS, formData);
  },
  destroy: (teamId: number) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}`;

    return sendRequest("DELETE", path);
  },
  load: (teamId: number | undefined): Promise<ILoadTeamResponse> => {
    if (!teamId || teamId <= API_NO_TEAM_ID) {
      return Promise.reject(
        new Error(
          `Invalid team id: ${teamId} must be greater than ${API_NO_TEAM_ID}`
        )
      );
    }
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}`;

    return sendRequest("GET", path);
  },
  loadAll: ({
    globalFilter = "",
  }: ILoadTeamsParams = {}): Promise<ILoadTeamsResponse> => {
    const queryParams = {
      query: globalFilter,
    };

    const queryString = buildQueryStringFromParams(queryParams);
    const endpoint = endpoints.TEAMS;
    const path = `${endpoint}?${queryString}`;

    return sendRequest("GET", path);
  },
  update: (
    {
      name,
      webhook_settings,
      integrations,
      mdm,
      host_expiry_settings,
    }: Partial<IUpdateTeamFormData>,
    teamId?: number
  ): Promise<ITeamConfig> => {
    if (typeof teamId === "undefined") {
      return Promise.reject("Invalid usage: missing team id");
    }

    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}`;
    const requestBody: Record<string, unknown> = {};
    if (name) {
      requestBody.name = name.trim();
    }
    if (webhook_settings) {
      requestBody.webhook_settings = webhook_settings;
    }
    if (integrations) {
      const { jira, zendesk, google_calendar } = integrations;
      const teamIntegrationProps = [
        "enable_failing_policies",
        "group_id",
        "project_key",
        "url",
      ];
      requestBody.integrations = {
        jira: jira?.map((j) => pick(j, teamIntegrationProps)),
        zendesk: zendesk?.map((z) => pick(z, teamIntegrationProps)),
        google_calendar,
      };
    }
    if (mdm) {
      requestBody.mdm = mdm;
    }
    if (host_expiry_settings) {
      requestBody.host_expiry_settings = host_expiry_settings;
    }

    return sendRequest("PATCH", path, requestBody);
  },

  /**
   * updates the team config. This can take any partial data that is in the team config.
   */
  updateConfig: (data: any, teamId: number): Promise<ITeamConfig> => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}`;
    return sendRequest("PATCH", path, data);
  },

  addUsers: (teamId: number | undefined, newUsers: INewTeamUsersBody) => {
    if (!teamId || teamId <= API_NO_TEAM_ID) {
      return Promise.reject(
        new Error(
          `Invalid team id: ${teamId} must be greater than ${API_NO_TEAM_ID}`
        )
      );
    }
    const { TEAM_USERS } = endpoints;
    const path = TEAM_USERS(teamId);

    return sendRequest("PATCH", path, newUsers);
  },
  removeUsers: (
    teamId: number | undefined,
    removeUsers: IRemoveTeamUserBody
  ) => {
    if (!teamId || teamId <= API_NO_TEAM_ID) {
      return Promise.reject(
        new Error(
          `Invalid team id: ${teamId} must be greater than ${API_NO_TEAM_ID}`
        )
      );
    }
    const { TEAM_USERS } = endpoints;
    const path = TEAM_USERS(teamId);

    return sendRequest("DELETE", path, removeUsers);
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
