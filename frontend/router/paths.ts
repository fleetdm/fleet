import URL_PREFIX from "./url_prefix";

const INTEGRATIONS_PREFIX = `${URL_PREFIX}/settings/integrations`;

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
  CONTROLS_VARIABLES: `${URL_PREFIX}/controls/variables`,

  // Dashboard pages
  DASHBOARD: `${URL_PREFIX}/dashboard`,
  DASHBOARD_LINUX: `${URL_PREFIX}/dashboard/linux`,
  DASHBOARD_MAC: `${URL_PREFIX}/dashboard/mac`,
  DASHBOARD_WINDOWS: `${URL_PREFIX}/dashboard/windows`,
  DASHBOARD_CHROME: `${URL_PREFIX}/dashboard/chrome`,
  DASHBOARD_IOS: `${URL_PREFIX}/dashboard/ios`,
  DASHBOARD_IPADOS: `${URL_PREFIX}/dashboard/ipados`,
  DASHBOARD_ANDROID: `${URL_PREFIX}/dashboard/android`,

  /**
   * Admin pages
   */

  ADMIN_SETTINGS: `${URL_PREFIX}/settings`,
  ADMIN_USERS: `${URL_PREFIX}/settings/users`,

  // Integrations pages

  ADMIN_INTEGRATIONS: INTEGRATIONS_PREFIX,
  ADMIN_INTEGRATIONS_TICKET_DESTINATIONS: `${INTEGRATIONS_PREFIX}/ticket-destinations`,
  ADMIN_INTEGRATIONS_MDM: `${INTEGRATIONS_PREFIX}/mdm`,
  ADMIN_INTEGRATIONS_MDM_APPLE: `${INTEGRATIONS_PREFIX}/mdm/apple`,
  ADMIN_INTEGRATIONS_MDM_WINDOWS: `${INTEGRATIONS_PREFIX}/mdm/windows`,
  ADMIN_INTEGRATIONS_MDM_ANDROID: `${INTEGRATIONS_PREFIX}/mdm/android`,
  ADMIN_INTEGRATIONS_APPLE_BUSINESS_MANAGER: `${INTEGRATIONS_PREFIX}/mdm/abm`,
  ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS: `${INTEGRATIONS_PREFIX}/automatic-enrollment/windows`,
  ADMIN_INTEGRATIONS_SCEP: `${INTEGRATIONS_PREFIX}/mdm/scep`,
  ADMIN_INTEGRATIONS_CALENDARS: `${INTEGRATIONS_PREFIX}/calendars`,
  ADMIN_INTEGRATIONS_CHANGE_MANAGEMENT: `${INTEGRATIONS_PREFIX}/change-management`,
  ADMIN_INTEGRATIONS_CONDITIONAL_ACCESS: `${INTEGRATIONS_PREFIX}/conditional-access`,
  ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES: `${INTEGRATIONS_PREFIX}/certificates`,
  ADMIN_INTEGRATIONS_IDENTITY_PROVIDER: `${INTEGRATIONS_PREFIX}/identity-provider`,
  ADMIN_INTEGRATIONS_VPP: `${INTEGRATIONS_PREFIX}/mdm/vpp`,
  ADMIN_INTEGRATIONS_VPP_SETUP: `${INTEGRATIONS_PREFIX}/vpp/setup`,
  ADMIN_INTEGRATIONS_SSO: `${INTEGRATIONS_PREFIX}/sso`,
  ADMIN_INTEGRATIONS_HOST_STATUS_WEBHOOK: `${INTEGRATIONS_PREFIX}/host-status-webhook`,

  ADMIN_TEAMS: `${URL_PREFIX}/settings/teams`,
  ADMIN_ORGANIZATION: `${URL_PREFIX}/settings/organization`,
  ADMIN_ORGANIZATION_INFO: `${URL_PREFIX}/settings/organization/info`,
  ADMIN_ORGANIZATION_WEBADDRESS: `${URL_PREFIX}/settings/organization/webaddress`,
  ADMIN_ORGANIZATION_SMTP: `${URL_PREFIX}/settings/organization/smtp`,
  ADMIN_ORGANIZATION_AGENTS: `${URL_PREFIX}/settings/organization/agents`,
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
  NEW_LABEL: `${URL_PREFIX}/labels/new`,
  // deprecated - now handled by `/new` route
  LABEL_NEW_DYNAMIC: `${URL_PREFIX}/labels/new/dynamic`,
  // deprecated - now handled by `/new` route
  LABEL_NEW_MANUAL: `${URL_PREFIX}/labels/new/manual`,

  LABEL_EDIT: (labelId: number) => `${URL_PREFIX}/labels/${labelId}`,

  EDIT_PACK: (packId: number): string => {
    return `${URL_PREFIX}/packs/${packId}/edit`;
  },
  PACK: (packId: number): string => `${URL_PREFIX}/packs/${packId}`,
  EDIT_LABEL: (labelId: number): string => `${URL_PREFIX}/labels/${labelId}`,
  EDIT_QUERY: (queryId: number): string =>
    `${URL_PREFIX}/queries/${queryId}/edit`,
  LIVE_QUERY: (queryId: number | null): string =>
    `${URL_PREFIX}/queries/${queryId || "new"}/live`,
  QUERY_DETAILS: (queryId: number): string =>
    `${URL_PREFIX}/queries/${queryId}`,
  EDIT_POLICY: (policyId: number): string =>
    `${URL_PREFIX}/policies/${policyId}`,
  FORGOT_PASSWORD: `${URL_PREFIX}/login/forgot`,
  MFA: `${URL_PREFIX}/login/mfa`,
  NO_ACCESS: `${URL_PREFIX}/login/denied`,
  API_ONLY_USER: `${URL_PREFIX}/apionlyuser`,

  // error pages
  FLEET_403: `${URL_PREFIX}/403`,
  FLEET_404: `${URL_PREFIX}/404`,
  FLEET_500: `${URL_PREFIX}/500`,

  LOGIN: `${URL_PREFIX}/login`,
  LOGOUT: `${URL_PREFIX}/logout`,
  MANAGE_HOSTS: `${URL_PREFIX}/hosts/manage`,
  MANAGE_HOSTS_LABEL: (labelId: number | string): string => {
    return `${URL_PREFIX}/hosts/manage/labels/${labelId}`;
  },
  HOST_DETAILS_PAGE: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}`;
  },
  HOST_DETAILS: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/details`;
  },
  HOST_SCRIPTS: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/scripts`;
  },
  HOST_SOFTWARE: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/software`;
  },
  HOST_INVENTORY: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/software/inventory`;
  },
  HOST_LIBRARY: (id: number): string => {
    return `${URL_PREFIX}/hosts/${id}/software/library`;
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
  MANAGE_POLICIES: `${URL_PREFIX}/policies/manage`,
  NEW_POLICY: `${URL_PREFIX}/policies/new`,
  NEW_QUERY: `${URL_PREFIX}/queries/new`,
  RESET_PASSWORD: `${URL_PREFIX}/login/reset`,
  SETUP: `${URL_PREFIX}/setup`,
  ACCOUNT: `${URL_PREFIX}/account`,
  URL_PREFIX,
};
