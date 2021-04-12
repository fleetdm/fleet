export default {
  CHANGE_PASSWORD: "/v1/fleet/change_password",
  CONFIG: "/v1/fleet/config",
  VERSION: "/v1/fleet/version",
  CONFIRM_EMAIL_CHANGE: (token) => {
    return `/v1/fleet/email/change/${token}`;
  },
  OSQUERY_OPTIONS: "/v1/fleet/spec/osquery_options",
  ENABLE_USER: (id) => {
    return `/v1/fleet/users/${id}/enable`;
  },
  FORGOT_PASSWORD: "/v1/fleet/forgot_password",
  HOSTS: "/v1/fleet/hosts",
  INVITES: "/v1/fleet/invites",
  LABELS: "/v1/fleet/labels",
  LABEL_HOSTS: (id) => {
    return `/v1/fleet/labels/${id}/hosts`;
  },
  LOGIN: "/v1/fleet/login",
  LOGOUT: "/v1/fleet/logout",
  ME: "/v1/fleet/me",
  PACKS: "/v1/fleet/packs",
  PERFORM_REQUIRED_PASSWORD_RESET: "/v1/fleet/perform_required_password_reset",
  QUERIES: "/v1/fleet/queries",
  RESET_PASSWORD: "/v1/fleet/reset_password",
  RUN_QUERY: "/v1/fleet/queries/run",
  SCHEDULED_QUERIES: "/v1/fleet/schedule",
  SCHEDULED_QUERY: (pack) => {
    return `/v1/fleet/packs/${pack.id}/scheduled`;
  },
  SETUP: "/v1/setup",
  STATUS_LABEL_COUNTS: "/v1/fleet/host_summary",
  TARGETS: "/v1/fleet/targets",
  USERS: "/v1/fleet/users",
  UPDATE_USER_ADMIN: (id) => {
    return `/v1/fleet/users/${id}/admin`;
  },
  SSO: "/v1/fleet/sso",
  STATUS_LIVE_QUERY: "/v1/fleet/status/live_query",
  STATUS_RESULT_STORE: "/v1/fleet/status/result_store",
};
