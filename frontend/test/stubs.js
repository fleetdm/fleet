export const adminUserStub = {
  id: 1,
  admin: true,
  email: 'hi@gnar.dog',
  name: 'Gnar Mike',
  username: 'gnardog',
};

export const configOptionStub = {
  id: 1,
  name: 'config_option_name',
  value: 'config option value',
  read_only: false,
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

export const hostStub = {
  created_at: '2017-01-10T19:18:55Z',
  updated_at: '2017-01-10T20:13:52Z',
  deleted_at: null,
  deleted: false,
  id: 1,
  detail_updated_at: '2017-01-10T20:01:48Z',
  seen_time: '2017-01-10T20:13:54Z',
  hostname: '52883a0ba916',
  uuid: 'FD87130B-09A9-683D-9095-D92CD20728CA',
  platform: 'ubuntu',
  osquery_version: '2.1.2',
  os_version: 'Ubuntu 14.4.',
  build: '',
  platform_like: 'debian',
  code_name: '',
  uptime: 45469000000000,
  memory: 2094940160,
  cpu_type: '6',
  cpu_subtype: '78',
  cpu_brand: 'Intel(R) Core(TM) i5-6267U CPU @ 2.90GHz',
  cpu_physical_cores: 2,
  cpu_logical_cores: 2,
  hardware_vendor: ' ',
  hardware_model: 'BHYVE',
  hardware_version: '1.0',
  hardware_serial: 'None',
  computer_name: '52883a0ba916',
  primary_ip_id: 1,
  network_interfaces: [
    {
      id: 1,
      interface: 'eth0',
      address: '172.19.0.8',
      mask: '255.255.0.0',
      broadcast: '172.19.0.8',
      point_to_point: '',
      mac: '02:42:ac:13:00:08',
      type: 1,
      mtu: 1500,
      metric: 0,
      ipackets: 512,
      opackets: 317,
      ibytes: 97231,
      obytes: 43502,
      ierrors: 0,
      oerrors: 0,
      last_change: -1,
    },
  ],
  status: 'online',
  display_text: '52883a0ba916',
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
  host_ids: [],
  label_ids: [],
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
  hostStub,
  packStub,
  queryStub,
  scheduledQueryStub,
  userStub,
};
