export default {
  ACTIVITIES: "/latest/fleet/activities",
  CHANGE_PASSWORD: "/latest/fleet/change_password",
  CONFIG: "/latest/fleet/config",
  CONFIRM_EMAIL_CHANGE: (token: string): string => {
    return `/latest/fleet/email/change/${token}`;
  },
  DEVICE_USER_DETAILS: "/latest/fleet/device",
  ENABLE_USER: (id: number): string => {
    return `/latest/fleet/users/${id}/enable`;
  },
  FORGOT_PASSWORD: "/latest/fleet/forgot_password",
  GLOBAL_ENROLL_SECRETS: "/latest/fleet/spec/enroll_secret",
  GLOBAL_POLICIES: "/latest/fleet/policies",
  GLOBAL_SCHEDULE: "/latest/fleet/schedule",
  HOST_SUMMARY: "/latest/fleet/host_summary",
  HOSTS: "/latest/fleet/hosts",
  HOSTS_COUNT: "/latest/fleet/hosts/count",
  HOSTS_DELETE: "/latest/fleet/hosts/delete",
  HOSTS_REPORT: "/latest/fleet/hosts/report",
  HOSTS_TRANSFER: "/latest/fleet/hosts/transfer",
  HOSTS_TRANSFER_BY_FILTER: "/latest/fleet/hosts/transfer/filter",
  INVITES: "/latest/fleet/invites",
  LABELS: "/latest/fleet/labels",
  LABEL_HOSTS: (id: number): string => {
    return `/latest/fleet/labels/${id}/hosts`;
  },
  LOGIN: "/latest/fleet/login",
  LOGOUT: "/latest/fleet/logout",
  MACADMINS: "/latest/fleet/macadmins",
  ME: "/latest/fleet/me",
  OS_VERSIONS: "/latest/fleet/os_versions",
  OSQUERY_OPTIONS: "/latest/fleet/spec/osquery_options",
  PACKS: "/latest/fleet/packs",
  PERFORM_REQUIRED_PASSWORD_RESET: "/latest/fleet/perform_required_password_reset",
  QUERIES: "/latest/fleet/queries",
  RESET_PASSWORD: "/latest/fleet/reset_password",
  RUN_QUERY: "/latest/fleet/queries/run",
  SCHEDULED_QUERIES: "/latest/fleet/schedule",
  SCHEDULED_QUERY: (id: number): string => {
    return `/latest/fleet/packs/${id}/scheduled`;
  },
  SETUP: "/latest/setup",
  SOFTWARE: "/latest/fleet/software",
  SSO: "/latest/fleet/sso",
  STATUS_LABEL_COUNTS: "/latest/fleet/host_summary",
  STATUS_LIVE_QUERY: "/latest/fleet/status/live_query",
  STATUS_RESULT_STORE: "/latest/fleet/status/result_store",
  TARGETS: "/latest/fleet/targets",
  TEAM_POLICIES: (teamId: number): string => {
    return `/latest/fleet/teams/${teamId}/policies`;
  },
  TEAM_SCHEDULE: (teamId: number): string => {
    return `/latest/fleet/teams/${teamId}/schedule`;
  },
  TEAMS: "/latest/fleet/teams",
  TEAMS_AGENT_OPTIONS: (teamId: number): string => {
    return `/latest/fleet/teams/${teamId}/agent_options`;
  },
  TEAMS_ENROLL_SECRETS: (teamId: number): string => {
    return `/latest/fleet/teams/${teamId}/secrets`;
  },
  TEAMS_MEMBERS: (teamId: number): string => {
    return `/latest/fleet/teams/${teamId}/users`;
  },
  TEAMS_TRANSFER_HOSTS: (teamId: number): string => {
    return `/latest/fleet/teams/${teamId}/hosts`;
  },
  UPDATE_USER_ADMIN: (id: number): string => {
    return `/latest/fleet/users/${id}/admin`;
  },
  USER_SESSIONS: (id: number): string => {
    return `/latest/fleet/users/${id}/sessions`;
  },
  USERS: "/latest/fleet/users",
  USERS_ADMIN: "/latest/fleet/users/admin",
  VERSION: "/latest/fleet/version",
};
