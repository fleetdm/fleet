import { IHost } from "../interfaces/host";
import { IQuery } from "../interfaces/query";
import { IPolicy } from "../interfaces/policy";
import URL_PREFIX from "./url_prefix";

export default {
  ROOT: `${URL_PREFIX}/`,
  HOME: `${URL_PREFIX}/dashboard`,
  ADMIN_USERS: `${URL_PREFIX}/settings/users`,
  ADMIN_SETTINGS: `${URL_PREFIX}/settings/organization`,
  ADMIN_TEAMS: `${URL_PREFIX}/settings/teams`,
  ALL_PACKS: `${URL_PREFIX}/packs/all`,
  EDIT_PACK: (packId: number): string => {
    return `${URL_PREFIX}/packs/${packId}/edit`;
  },
  PACK: (packId: number): string => {
    return `${URL_PREFIX}/packs/${packId}`;
  },
  EDIT_QUERY: (query: IQuery): string => {
    return `${URL_PREFIX}/queries/${query.id}`;
  },
  EDIT_POLICY: (policy: IPolicy): string => {
    return `${URL_PREFIX}/policies/${policy.id}${
      policy.team_id ? `?team_id=${policy.team_id}` : ""
    }`;
  },
  FORGOT_PASSWORD: `${URL_PREFIX}/login/forgot`,
  API_ONLY_USER: `${URL_PREFIX}/apionlyuser`,
  FLEET_403: `${URL_PREFIX}/403`,
  // FLEET_500: `${URL_PREFIX}/500`,
  LOGIN: `${URL_PREFIX}/login`,
  LOGOUT: `${URL_PREFIX}/logout`,
  MANAGE_HOSTS: `${URL_PREFIX}/hosts/manage`,
  HOST_DETAILS: (host: IHost): string => {
    return `${URL_PREFIX}/hosts/${host.id}`;
  },
  TEAM_DETAILS_MEMBERS: (teamId: number): string => {
    return `${URL_PREFIX}/settings/teams/${teamId}/members`;
  },
  TEAM_DETAILS_OPTIONS: (teamId: number): string => {
    return `${URL_PREFIX}/settings/teams/${teamId}/options`;
  },
  MANAGE_PACKS: `${URL_PREFIX}/packs/manage`,
  NEW_PACK: `${URL_PREFIX}/packs/new`,
  MANAGE_QUERIES: `${URL_PREFIX}/queries/manage`,
  MANAGE_SCHEDULE: `${URL_PREFIX}/schedule/manage`,
  MANAGE_TEAM_SCHEDULE: (teamId: number): string => {
    return `${URL_PREFIX}/schedule/manage/teams/${teamId}`;
  },
  MANAGE_POLICIES: `${URL_PREFIX}/policies/manage`,
  NEW_POLICY: `${URL_PREFIX}/policies/new`,
  NEW_QUERY: `${URL_PREFIX}/queries/new`,
  RESET_PASSWORD: `${URL_PREFIX}/login/reset`,
  SETUP: `${URL_PREFIX}/setup`,
  USER_SETTINGS: `${URL_PREFIX}/profile`,
  URL_PREFIX,
};
