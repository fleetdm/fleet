export const adminUserStub = {
  id: 1,
  admin: true,
  email: 'hi@gnar.dog',
  name: 'Gnar Mike',
  username: 'gnardog',
};

export const configStub = {
  org_info: {
    org_name: 'Kolide',
    org_logo_url: '0.0.0.0:8080/logo.png',
  },
  server_settings: {
    kolide_server_url: '',
  },
  smtp_settings: {
    configured: false,
    sender_address: '',
    server: '',
    port: 587,
    authentication_type: 'authtype_username_password',
    user_name: '',
    password: '',
    enable_ssl_tls: true,
    authentication_method: 'authmethod_plain',
    verify_ssl_certs: true,
    enable_start_tls: true,
    email_enabled: false,
  },
};

export const packStub = {
  created_at: '0001-01-01T00:00:00Z',
  updated_at: '0001-01-01T00:00:00Z',
  deleted_at: null,
  deleted: false,
  id: 3,
  name: 'Pack Name',
  description: 'Pack Description',
  platform: '',
  created_by: 1,
  disabled: false,
};

export const queryStub = {
  created_at: '2016-10-17T07:06:00Z',
  deleted: false,
  deleted_at: null,
  description: '',
  differential: false,
  id: 1,
  interval: 0,
  name: 'dev_query_1',
  platform: '',
  query: 'select * from processes',
  snapshot: false,
  updated_at: '2016-10-17T07:06:00Z',
  version: '',
};

export const scheduledQueryStub = {
  id: 1,
  interval: 60,
  name: 'Get all users',
  pack_id: 123,
  platform: 'darwin',
  query: 'SELECT * FROM users',
  query_id: 5,
  removed: false,
  snapshot: true,
};

export const userStub = {
  id: 1,
  admin: false,
  email: 'hi@gnar.dog',
  name: 'Gnar Mike',
  username: 'gnardog',
};

export default {
  adminUserStub,
  configStub,
  packStub,
  queryStub,
  scheduledQueryStub,
  userStub,
};
