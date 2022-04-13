const API_VERSION = "latest";

export default {
  ACTIVITIES: `/${API_VERSION}/fleet/activities`,
  CHANGE_PASSWORD: `/${API_VERSION}/fleet/change_password`,
  CONFIG: `/${API_VERSION}/fleet/config`,
  CONFIRM_EMAIL_CHANGE: (token: string): string => {
    return `/${API_VERSION}/fleet/email/change/${token}`;
  },
  DEVICE_USER_DETAILS: `/${API_VERSION}/fleet/device`,
  ENABLE_USER: (id: number): string => {
    return `/${API_VERSION}/fleet/users/${id}/enable`;
  },
  FORGOT_PASSWORD: `/${API_VERSION}/fleet/forgot_password`,
  GLOBAL_ENROLL_SECRETS: `/${API_VERSION}/fleet/spec/enroll_secret`,
  GLOBAL_POLICIES: `/${API_VERSION}/fleet/policies`,
  GLOBAL_SCHEDULE: `/${API_VERSION}/fleet/schedule`,
  HOST_SUMMARY: `/${API_VERSION}/fleet/host_summary`,
  HOSTS: `/${API_VERSION}/fleet/hosts`,
  HOSTS_COUNT: `/${API_VERSION}/fleet/hosts/count`,
  HOSTS_DELETE: `/${API_VERSION}/fleet/hosts/delete`,
  HOSTS_REPORT: `/${API_VERSION}/fleet/hosts/report`,
  HOSTS_TRANSFER: `/${API_VERSION}/fleet/hosts/transfer`,
  HOSTS_TRANSFER_BY_FILTER: `/${API_VERSION}/fleet/hosts/transfer/filter`,
  INVITES: `/${API_VERSION}/fleet/invites`,
  LABELS: `/${API_VERSION}/fleet/labels`,
  LABEL_HOSTS: (id: number): string => {
    return `/${API_VERSION}/fleet/labels/${id}/hosts`;
  },
  LOGIN: `/${API_VERSION}/fleet/login`,
  LOGOUT: `/${API_VERSION}/fleet/logout`,
  MACADMINS: `/${API_VERSION}/fleet/macadmins`,
  ME: `/${API_VERSION}/fleet/me`,
  OS_VERSIONS: `/${API_VERSION}/fleet/os_versions`,
  OSQUERY_OPTIONS: `/${API_VERSION}/fleet/spec/osquery_options`,
  PACKS: `/${API_VERSION}/fleet/packs`,
  PERFORM_REQUIRED_PASSWORD_RESET: `/${API_VERSION}/fleet/perform_required_password_reset`,
  QUERIES: `/${API_VERSION}/fleet/queries`,
  RESET_PASSWORD: `/${API_VERSION}/fleet/reset_password`,
  RUN_QUERY: `/${API_VERSION}/fleet/queries/run`,
  SCHEDULE_QUERY: `/${API_VERSION}/fleet/packs/schedule`,
  SCHEDULED_QUERIES: (packId: number): string => {
    return `/${API_VERSION}/fleet/packs/${packId}/scheduled`;
  },
  SETUP: `/v1/setup`, // not a typo - hasn't been updated yet
  SOFTWARE: `/${API_VERSION}/fleet/software`,
  SSO: `/${API_VERSION}/fleet/sso`,
  STATUS_LABEL_COUNTS: `/${API_VERSION}/fleet/host_summary`,
  STATUS_LIVE_QUERY: `/${API_VERSION}/fleet/status/live_query`,
  STATUS_RESULT_STORE: `/${API_VERSION}/fleet/status/result_store`,
  TARGETS: `/${API_VERSION}/fleet/targets`,
  TEAM_POLICIES: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/policies`;
  },
  TEAM_SCHEDULE: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/schedule`;
  },
  TEAMS: `/${API_VERSION}/fleet/teams`,
  TEAMS_AGENT_OPTIONS: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/agent_options`;
  },
  TEAMS_ENROLL_SECRETS: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/secrets`;
  },
  TEAMS_MEMBERS: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/users`;
  },
  TEAMS_TRANSFER_HOSTS: (teamId: number): string => {
    return `/${API_VERSION}/fleet/teams/${teamId}/hosts`;
  },
  UPDATE_USER_ADMIN: (id: number): string => {
    return `/${API_VERSION}/fleet/users/${id}/admin`;
  },
  USER_SESSIONS: (id: number): string => {
    return `/${API_VERSION}/fleet/users/${id}/sessions`;
  },
  USERS: `/${API_VERSION}/fleet/users`,
  USERS_ADMIN: `/${API_VERSION}/fleet/users/admin`,
  VERSION: `/${API_VERSION}/fleet/version`,
};
