export default {
  ADMIN_DASHBOARD: '/admin',
  ADMIN_SETTINGS: '/admin/settings',
  ALL_PACKS: '/packs/all',
  EDIT_QUERY: (query) => {
    return `/queries/${query.id}`;
  },
  FORGOT_PASSWORD: '/login/forgot',
  HOME: '/',
  KOLIDE_500: '/500',
  LICENSE: '/license',
  LOGIN: '/login',
  LOGOUT: '/logout',
  MANAGE_HOSTS: '/hosts/manage',
  NEW_PACK: '/packs/new',
  NEW_QUERY: '/queries/new',
  RESET_PASSWORD: '/login/reset',
  SETUP: '/setup',
};
