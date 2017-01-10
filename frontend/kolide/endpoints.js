export default {
  CONFIG: '/v1/kolide/config',
  FORGOT_PASSWORD: '/v1/kolide/forgot_password',
  HOSTS: '/v1/kolide/hosts',
  INVITES: '/v1/kolide/invites',
  LABELS: '/v1/kolide/labels',
  LABEL_HOSTS: (id) => {
    return `/v1/kolide/labels/${id}/hosts`;
  },
  LOGIN: '/v1/kolide/login',
  LOGOUT: '/v1/kolide/logout',
  ME: '/v1/kolide/me',
  PACKS: '/v1/kolide/packs',
  PERFORM_REQUIRED_PASSWORD_RESET: '/v1/kolide/perform_required_password_reset',
  QUERIES: '/v1/kolide/queries',
  RESET_PASSWORD: '/v1/kolide/reset_password',
  RUN_QUERY: '/v1/kolide/queries/run',
  SCHEDULED_QUERIES: (pack) => {
    return `/v1/kolide/packs/${pack.id}/scheduled`;
  },
  SETUP: '/v1/setup',
  TARGETS: '/v1/kolide/targets',
  USERS: '/v1/kolide/users',
};
