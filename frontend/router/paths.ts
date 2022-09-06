import { IHost } from "../interfaces/host";
import { IQuery } from "../interfaces/query";
import { IPolicy } from "../interfaces/policy";
import URL_PREFIX from "./url_prefix";

export default {
  ROOT: `${URL_PREFIX}/`,
  HOME: `${URL_PREFIX}/dashboard`,
  ADMIN_USERS: `${URL_PREFIX}/settings/users`,
  ADMIN_INTEGRATIONS: `${URL_PREFIX}/settings/integrations`,
  ADMIN_TEAMS: `${URL_PREFIX}/settings/teams`,
  ADMIN_SETTINGS: `${URL_PREFIX}/settings/organization`,
  ADMIN_SETTINGS_INFO: `${URL_PREFIX}/settings/organization/info`,
  ADMIN_SETTINGS_WEBADDRESS: `${URL_PREFIX}/settings/organization/webaddress`,
  ADMIN_SETTINGS_SSO: `${URL_PREFIX}/settings/organization/sso`,
  ADMIN_SETTINGS_SMTP: `${URL_PREFIX}/settings/organization/smtp`,
  ADMIN_SETTINGS_AGENTS: `${URL_PREFIX}/settings/organization/agents`,
  ADMIN_SETTINGS_HOST_STATUS_WEBHOOK: `${URL_PREFIX}/settings/organization/host-status-webhook`,
  ADMIN_SETTINGS_STATISTICS: `${URL_PREFIX}/settings/organization/statistics`,
  ADMIN_SETTINGS_ADVANCED: `${URL_PREFIX}/settings/organization/advanced`,
  ADMIN_SETTINGS_FLEET_DESKTOP: `${URL_PREFIX}/settings/organization/fleet-desktop`,
  EDIT_PACK: (packId: number): string => {
    return `${URL_PREFIX}/packs/${packId}/edit`;
  },
  PACK: (packId: number): string => {
    return `${URL_PREFIX}/packs/${packId}`;
  },
  EDIT_LABEL: (labelId: number): string => {
    return `${URL_PREFIX}/labels/${labelId}`;
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
  LOGIN: `${URL_PREFIX}/login`,
  LOGOUT: `${URL_PREFIX}/logout`,
  MANAGE_HOSTS: `${URL_PREFIX}/hosts/manage`,
  MANAGE_HOSTS_LABEL: (labelId: number | string): string => {
    return `${URL_PREFIX}/hosts/manage/labels/${labelId}`;
  },
  HOST_DETAILS: (host: IHost): string => {
    return `${URL_PREFIX}/hosts/${host.id}`;
  },
  DEVICE_USER_DETAILS: (deviceAuthToken: any): string => {
    return `${URL_PREFIX}/device/${deviceAuthToken}`;
  },
  MANAGE_SOFTWARE: `${URL_PREFIX}/software/manage`,
  SOFTWARE_DETAILS: (id: string): string => {
    return `${URL_PREFIX}/software/${id}`;
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
  NEW_LABEL: `${URL_PREFIX}/labels/new`,
  NEW_POLICY: `${URL_PREFIX}/policies/new`,
  NEW_QUERY: `${URL_PREFIX}/queries/new`,
  RESET_PASSWORD: `${URL_PREFIX}/login/reset`,
  SETUP: `${URL_PREFIX}/setup`,
  USER_SETTINGS: `${URL_PREFIX}/profile`,
  URL_PREFIX,
};
