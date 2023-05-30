import { IConfig } from "interfaces/config";

const DEFAULT_CONFIG_MOCK: IConfig = {
  org_info: {
    org_name: "fleet",
    org_logo_url: "",
  },
  server_settings: {
    server_url: "https://localhost:8080",
    live_query_disabled: false,
    enable_analytics: true,
  },
  smtp_settings: {
    enable_smtp: false,
    configured: false,
    sender_address: "",
    server: "",
    port: 587,
    authentication_type: "authtype_username_password",
    user_name: "",
    password: "********",
    enable_ssl_tls: true,
    authentication_method: "authmethod_plain",
    domain: "",
    verify_ssl_certs: true,
    enable_start_tls: true,
  },
  sso_settings: {
    entity_id: "",
    idp_image_url: "",
    metadata: "",
    metadata_url: "",
    idp_name: "",
    enable_sso: false,
    enable_sso_idp_login: false,
    enable_jit_provisioning: false,
  },
  host_expiry_settings: {
    host_expiry_enabled: false,
    host_expiry_window: 0,
  },
  agent_options: "",
  license: {
    tier: "free",
    expiration: "0001-01-01T00:00:00Z",
    device_count: 4,
    note: "",
    organization: "",
  },
  webhook_settings: {
    host_status_webhook: {
      enable_host_status_webhook: true,
      destination_url: "https://server.com",
      host_percentage: 5,
      days_count: 7,
    },
    failing_policies_webhook: {
      enable_failing_policies_webhook: true,
      destination_url: "https://server.com",
      policy_ids: [1, 2, 3],
      host_batch_size: 1000,
    },
    vulnerabilities_webhook: {
      enable_vulnerabilities_webhook: true,
      destination_url: "https://server.com",
      host_batch_size: 1000,
    },
  },
  integrations: {
    jira: [],
    zendesk: [],
  },
  logging: {
    debug: false,
    json: false,
    result: {
      plugin: "filesystem",
      config: {
        status_log_file:
          "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_status",
        result_log_file:
          "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_result",
        enable_log_rotation: false,
        enable_log_compression: false,
      },
    },
    status: {
      plugin: "filesystem",
      config: {
        status_log_file:
          "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_status",
        result_log_file:
          "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_result",
        enable_log_rotation: false,
        enable_log_compression: false,
      },
    },
  },
  update_interval: {
    osquery_detail: 3600000000000,
    osquery_policy: 3600000000000,
  },
  vulnerabilities: {
    cpe_database_url: "",
    current_instance_checks: "auto",
    cve_feed_prefix_url: "",
    databases_path: "",
    disable_data_sync: false,
    periodicity: 3600000000000,
    recent_vulnerability_max_age: 2592000000000000,
  },
};

const createMockConfig = (overrides?: Partial<IConfig>): IConfig => {
  return { ...DEFAULT_CONFIG_MOCK, ...overrides };
};

export default createMockConfig;
