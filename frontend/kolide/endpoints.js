export default {
  CHANGE_PASSWORD: '/v1/kolide/change_password',
  CONFIG: '/v1/kolide/config',
  CONFIG_OPTIONS: '/v1/kolide/options',
  CONFIG_OPTIONS_RESET: '/v1/kolide/options/reset',
  CONFIRM_EMAIL_CHANGE: (token) => {
    return `/v1/kolide/email/change/${token}`;
  },
  ENABLE_USER: (id) => {
    return `/v1/kolide/users/${id}/enable`;
  },
  FORGOT_PASSWORD: '/v1/kolide/forgot_password',
  HOSTS: '/v1/kolide/hosts',
  INVITES: '/v1/kolide/invites',
  LABELS: '/v1/kolide/labels',
  LABEL_HOSTS: (id) => {
    return `/v1/kolide/labels/${id}/hosts`;
  },
  LICENSE: '/v1/kolide/license',
  LOGIN: '/v1/kolide/login',
  LOGOUT: '/v1/kolide/logout',
  ME: '/v1/kolide/me',
  PACKS: '/v1/kolide/packs',
  PERFORM_REQUIRED_PASSWORD_RESET: '/v1/kolide/perform_required_password_reset',
  QUERIES: '/v1/kolide/queries',
  RESET_PASSWORD: '/v1/kolide/reset_password',
  RUN_QUERY: '/v1/kolide/queries/run',
  SCHEDULED_QUERIES: '/v1/kolide/schedule',
  SCHEDULED_QUERY: (pack) => {
    return `/v1/kolide/packs/${pack.id}/scheduled`;
  },
  SETUP: '/v1/setup',
  SETUP_LICENSE: '/v1/license',
  STATUS_LABEL_COUNTS: '/v1/kolide/host_summary',
  TARGETS: '/v1/kolide/targets',
  USERS: '/v1/kolide/users',
  UPDATE_USER_ADMIN: (id) => {
    return `/v1/kolide/users/${id}/admin`;
  },
};
