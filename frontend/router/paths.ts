import { buildQueryStringFromParams } from "utilities/url";

import { IPolicy } from "../interfaces/policy";
import URL_PREFIX from "./url_prefix";

// Note: changes to paths.ts should change page_titles.ts respectively
export default {
  ROOT: `${URL_PREFIX}/`,

  // Controls pages
  CONTROLS: `${URL_PREFIX}/controls`,
  CONTROLS_OS_UPDATES: `${URL_PREFIX}/controls/os-updates`,
  CONTROLS_OS_SETTINGS: `${URL_PREFIX}/controls/os-settings`,
  CONTROLS_CUSTOM_SETTINGS: `${URL_PREFIX}/controls/os-settings/custom-settings`,
  CONTROLS_DISK_ENCRYPTION: `${URL_PREFIX}/controls/os-settings/disk-encryption`,
  CONTROLS_SETUP_EXPERIENCE: `${URL_PREFIX}/controls/setup-experience`,
  CONTROLS_END_USER_AUTHENTICATION: `${URL_PREFIX}/controls/setup-experience/end-user-auth`,
  CONTROLS_BOOTSTRAP_PACKAGE: `${URL_PREFIX}/controls/setup-experience/bootstrap-package`,
  CONTROLS_SETUP_ASSITANT: `${URL_PREFIX}/controls/setup-experience/setup-assistant`,
  CONTROLS_INSTALL_SOFTWARE: `${URL_PREFIX}/controls/setup-experience/install-software`,
  CONTROLS_RUN_SCRIPT: `${URL_PREFIX}/controls/setup-experience/run-script`,
  CONTROLS_SCRIPTS: `${URL_PREFIX}/controls/scripts`,

  // Dashboard pages
  DASHBOARD: `${URL_PREFIX}/dashboard`,
  DASHBOARD_LINUX: `${URL_PREFIX}/dashboard/linux`,
  DASHBOARD_MAC: `${URL_PREFIX}/dashboard/mac`,
  DASHBOARD_WINDOWS: `${URL_PREFIX}/dashboard/windows`,
  DASHBOARD_CHROME: `${URL_PREFIX}/dashboard/chrome`,
  DASHBOARD_IOS: `${URL_PREFIX}/dashboard/ios`,
  DASHBOARD_IPADOS: `${URL_PREFIX}/dashboard/ipados`,

  /**
   * Admin pages
   */

  ADMIN_SETTINGS: `${URL_PREFIX}/settings`,
  ADMIN_USERS: `${URL_PREFIX}/settings/users`,

  // Integrations pages
  ADMIN_INTEGRATIONS: `${URL_PREFIX}/settings/integrations`,
  ADMIN_INTEGRATIONS_TICKET_DESTINATIONS: `${URL_PREFIX}/settings/integrations/ticket-destinations`,
  ADMIN_INTEGRATIONS_MDM: `${URL_PREFIX}/settings/integrations/mdm`,
  ADMIN_INTEGRATIONS_MDM_APPLE: `${URL_PREFIX}/settings/integrations/mdm/apple`,
  ADMIN_INTEGRATIONS_MDM_WINDOWS: `${URL_PREFIX}/settings/integrations/mdm/windows`,
  ADMIN_INTEGRATIONS_APPLE_BUSINESS_MANAGER: `${URL_PREFIX}/settings/integrations/mdm/abm`,
  ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS: `${URL_PREFIX}/settings/integrations/automatic-enrollment/windows`,
  ADMIN_INTEGRATIONS_SCEP: `${URL_PREFIX}/settings/integrations/mdm/scep`,
  ADMIN_INTEGRATIONS_CALENDARS: `${URL_PREFIX}/settings/integrations/calendars`,
  ADMIN_INTEGRATIONS_VPP: `${URL_PREFIX}/settings/integrations/mdm/vpp`,
  ADMIN_INTEGRATIONS_VPP_SETUP: `${URL_PREFIX}/settings/integrations/vpp/setup`,

  ADMIN_TEAMS: `${URL_PREFIX}/settings/teams`,
  ADMIN_ORGANIZATION: `${URL_PREFIX}/settings/organization`,
  ADMIN_ORGANIZATION_INFO: `${URL_PREFIX}/settings/organization/info`,
  ADMIN_ORGANIZATION_WEBADDRESS: `${URL_PREFIX}/settings/organization/webaddress`,
  ADMIN_ORGANIZATION_SSO: `${URL_PREFIX}/settings/organization/sso`,
  ADMIN_ORGANIZATION_SMTP: `${URL_PREFIX}/settings/organization/smtp`,
  ADMIN_ORGANIZATION_AGENTS: `${URL_PREFIX}/settings/organization/agents`,
  ADMIN_ORGANIZATION_HOST_STATUS_WEBHOOK: `${URL_PREFIX}/settings/organization/host-status-webhook`,
  ADMIN_ORGANIZATION_STATISTICS: `${URL_PREFIX}/settings/organization/statistics`,
  ADMIN_ORGANIZATION_ADVANCED: `${URL_PREFIX}/settings/organization/advanced`,
  ADMIN_ORGANIZATION_FLEET_DESKTOP: `${URL_PREFIX}/settings/organization/fleet-desktop`,

  // Software pages
  SOFTWARE: `${URL_PREFIX}/software`,
  SOFTWARE_TITLES: `${URL_PREFIX}/software/titles`,
  SOFTWARE_OS: `${URL_PREFIX}/software/os`,
  SOFTWARE_VERSIONS: `${URL_PREFIX}/software/versions`,
  SOFTWARE_TITLE_DETAILS: (id: string): string => {
    return `${URL_PREFIX}/software/titles/${id}`;
  },
  SOFTWARE_VERSION_DETAILS: (id: string): string => {
    return `${URL_PREFIX}/software/versions/${id}`;
  },
  SOFTWARE_OS_DETAILS: (id: number): string => {
    return `${URL_PREFIX}/software/os/${id}`;
  },
  SOFTWARE_VULNERABILITIES: `${URL_PREFIX}/software/vulnerabilities`,
  SOFTWARE_VULNERABILITY_DETAILS: (cve: string): string => {
    return `${URL_PREFIX}/software/vulnerabilities/${cve}`;
  },
  SOFTWARE_ADD_FLEET_MAINTAINED: `${URL_PREFIX}/software/add/fleet-maintained`,
  SOFTWARE_FLEET_MAINTAINED_DETAILS: (id: number) =>
    `${URL_PREFIX}/software/add/fleet-maintained/${id}`,
  SOFTWARE_ADD_PACKAGE: `${URL_PREFIX}/software/add/package`,
  SOFTWARE_ADD_APP_STORE: `${URL_PREFIX}/software/add/app-store`,

  // Label pages
  LABEL_NEW_DYNAMIC: `${URL_PREFIX}/labels/new/dynamic`,
  LABEL_NEW_MANUAL: `${URL_PREFIX}/labels/new/manual`,
  LABEL_EDIT: (labelId: number) => `${URL_PREFIX}/labels/${labelId}`,

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
    return `${URL_PREFIX}/queries/${queryId}/edit${
      teamId ? `?team_id=${teamId}` : ""
    }`;
  },
  LIVE_QUERY: (
    queryId: number | null,
    teamId?: number,
    hostId?: number
  ): string => {
    const baseUrl = `${URL_PREFIX}/queries/${queryId || "new"}/live`;
    const queryParams = buildQueryStringFromParams({
      team_id: teamId,
      host_id: hostId,
    });
    return queryParams ? `${baseUrl}?${queryParams}` : baseUrl;
  },
  QUERY_DETAILS: (queryId: number, teamId?: number): string => {
    return `${URL_PREFIX}/queries/${queryId}${
      teamId ? `?team_id=${teamId}` : ""
    }`;
  },
  EDIT_POLICY: (policy: IPolicy): string => {
    return `${URL_PREFIX}/policies/${policy.id}${
      policy.team_id !== undefined ? `?team_id=${policy.team_id}` : ""
    }`;
  },
  FORGOT_PASSWORD: `${URL_PREFIX}/login/forgot`,
  TWO_FACTOR_AUTHENTICATION: `${URL_PREFIX}/login/2fa`,
  EXPIRED: `${URL_PREFIX}/login/expired`,
  NO_ACCESS: `${URL_PREFIX}/login/denied`,
  API_ONLY_USER: `${URL_PREFIX}/apionlyuser`,

  // error pages
  FLEET_403: `${URL_PREFIX}/403`,
  FLEET_404: `${URL_PREFIX}/404`,
  FLEET_500: `${URL_PREFIX}/500`,

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
  HOST_SCRIPTS: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/scripts`;
  },
  HOST_SOFTWARE: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/software`;
  },
  HOST_QUERIES: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/queries`;
  },
  HOST_POLICIES: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/policies`;
  },
  HOST_QUERY_REPORT: (hostId: number, queryId: number): string =>
    `${URL_PREFIX}/hosts/${hostId}/queries/${queryId}`,
  DEVICE_USER_DETAILS: (deviceAuthToken: string): string => {
    return `${URL_PREFIX}/device/${deviceAuthToken}`;
  },
  DEVICE_USER_DETAILS_SELF_SERVICE: (deviceAuthToken: string): string => {
    return `${URL_PREFIX}/device/${deviceAuthToken}/self-service`;
  },
  DEVICE_USER_DETAILS_SOFTWARE: (deviceAuthToken: string): string => {
    return `${URL_PREFIX}/device/${deviceAuthToken}/software`;
  },
  DEVICE_USER_DETAILS_POLICIES: (deviceAuthToken: string): string => {
    return `${URL_PREFIX}/device/${deviceAuthToken}/policies`;
  },

  TEAM_DETAILS_USERS: (teamId?: number): string => {
    if (teamId !== undefined && teamId > 0) {
      return `${URL_PREFIX}/settings/teams/users?team_id=${teamId}`;
    }
    return `${URL_PREFIX}/settings/teams`;
  },
  TEAM_DETAILS_OPTIONS: (teamId?: number): string => {
    if (teamId !== undefined && teamId > 0) {
      return `${URL_PREFIX}/settings/teams/options?team_id=${teamId}`;
    }
    return `${URL_PREFIX}/settings/teams`;
  },
  TEAM_DETAILS_SETTINGS: (teamId?: number) => {
    if (teamId !== undefined && teamId > 0) {
      return `${URL_PREFIX}/settings/teams/settings?team_id=${teamId}`;
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
  ACCOUNT: `${URL_PREFIX}/account`,
  URL_PREFIX,
};
