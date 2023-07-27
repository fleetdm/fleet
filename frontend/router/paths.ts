import { IPolicy } from "../interfaces/policy";
import URL_PREFIX from "./url_prefix";

export default {
  ROOT: `${URL_PREFIX}/`,
  CONTROLS: `${URL_PREFIX}/controls`,
  CONTROLS_MAC_OS_UPDATES: `${URL_PREFIX}/controls/mac-os-updates`,
  CONTROLS_MAC_SETTINGS: `${URL_PREFIX}/controls/mac-settings`,
  CONTROLS_CUSTOM_SETTINGS: `${URL_PREFIX}/controls/mac-settings/custom-settings`,
  CONTROLS_DISK_ENCRYPTION: `${URL_PREFIX}/controls/mac-settings/disk-encryption`,
  CONTROLS_MAC_SETUP: `${URL_PREFIX}/controls/mac-setup`,
  CONTROLS_END_USER_AUTHENTICATION: `${URL_PREFIX}/controls/mac-setup/end-user-auth`,
  CONTROLS_BOOTSTRAP_PACKAGE: `${URL_PREFIX}/controls/mac-setup/bootstrap-package`,
  CONTROLS_MAC_SCRIPTS: `${URL_PREFIX}/controls/mac-scripts`,
  DASHBOARD: `${URL_PREFIX}/dashboard`,
  DASHBOARD_LINUX: `${URL_PREFIX}/dashboard/linux`,
  DASHBOARD_MAC: `${URL_PREFIX}/dashboard/mac`,
  DASHBOARD_WINDOWS: `${URL_PREFIX}/dashboard/windows`,
  DASHBOARD_CHROME: `${URL_PREFIX}/dashboard/chrome`,
  ADMIN_USERS: `${URL_PREFIX}/settings/users`,
  ADMIN_INTEGRATIONS: `${URL_PREFIX}/settings/integrations`,
  ADMIN_INTEGRATIONS_TICKET_DESTINATIONS: `${URL_PREFIX}/settings/integrations/ticket-destinations`,
  ADMIN_INTEGRATIONS_MDM: `${URL_PREFIX}/settings/integrations/mdm`,
  ADMIN_INTEGRATIONS_MDM_MAC: `${URL_PREFIX}/settings/integrations/mdm/apple`,
  ADMIN_INTEGRATIONS_MDM_WINDOWS: `${URL_PREFIX}/settings/integrations/mdm/windows`,
  ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT: `${URL_PREFIX}/settings/integrations/automatic-enrollment`,
  ADMIN_TEAMS: `${URL_PREFIX}/settings/teams`,
  ADMIN_SETTINGS: `${URL_PREFIX}/settings`,
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
  EDIT_QUERY: (queryId: number, teamId?: number): string => {
    return `${URL_PREFIX}/queries/${queryId}${
      teamId ? `?team_id=${teamId}` : ""
    }`;
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
  MANAGE_HOSTS_ADD_HOSTS: `${URL_PREFIX}/hosts/manage/?add_hosts=true`,
  MANAGE_HOSTS_LABEL: (labelId: number | string): string => {
    return `${URL_PREFIX}/hosts/manage/labels/${labelId}`;
  },
  HOST_DETAILS: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}`;
  },
  HOST_SOFTWARE: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/software`;
  },
  HOST_SCHEDULE: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/schedule`;
  },
  HOST_POLICIES: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/policies`;
  },
  DEVICE_USER_DETAILS: (deviceAuthToken: any): string => {
    return `${URL_PREFIX}/device/${deviceAuthToken}`;
  },
  DEVICE_USER_DETAILS_SOFTWARE: (deviceAuthToken: string): string => {
    return `${URL_PREFIX}/device/${deviceAuthToken}/software`;
  },
  DEVICE_USER_DETAILS_POLICIES: (deviceAuthToken: string): string => {
    return `${URL_PREFIX}/device/${deviceAuthToken}/policies`;
  },
  MANAGE_SOFTWARE: `${URL_PREFIX}/software/manage`,
  SOFTWARE_DETAILS: (id: string): string => {
    return `${URL_PREFIX}/software/${id}`;
  },
  TEAM_DETAILS_MEMBERS: (teamId?: number): string => {
    if (teamId !== undefined && teamId > 0) {
      return `${URL_PREFIX}/settings/teams/members?team_id=${teamId}`;
    }
    return `${URL_PREFIX}/settings/teams`;
  },
  TEAM_DETAILS_OPTIONS: (teamId?: number): string => {
    if (teamId !== undefined && teamId > 0) {
      return `${URL_PREFIX}/settings/teams/options?team_id=${teamId}`;
    }
    return `${URL_PREFIX}/settings/teams`;
  },
  MANAGE_PACKS: `${URL_PREFIX}/packs/manage`,
  NEW_PACK: `${URL_PREFIX}/packs/new`,
  MANAGE_QUERIES: `${URL_PREFIX}/queries/manage`,
  MANAGE_SCHEDULE: `${URL_PREFIX}/schedule/manage`,
  MANAGE_TEAM_SCHEDULE: (teamId: number): string => {
    return `${URL_PREFIX}/schedule/manage?team_id=${teamId}`;
  },
  MANAGE_POLICIES: `${URL_PREFIX}/policies/manage`,
  NEW_LABEL: `${URL_PREFIX}/labels/new`,
  NEW_POLICY: `${URL_PREFIX}/policies/new`,
  NEW_QUERY: (teamId?: number) =>
    `${URL_PREFIX}/queries/new${teamId ? `?team_id=${teamId}` : ""}`,
  RESET_PASSWORD: `${URL_PREFIX}/login/reset`,
  SETUP: `${URL_PREFIX}/setup`,
  USER_SETTINGS: `${URL_PREFIX}/profile`,
  URL_PREFIX,
};
