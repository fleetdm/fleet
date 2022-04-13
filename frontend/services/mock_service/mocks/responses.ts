/*
 * NOTE: This is an example of how to define data for your mock responses.
 * Be sure to copy this file into `../mocks` and only edit that copy!
 * Also please check the README for how to use the mock service :)
 */

const mockJira0 = {
  url: "https://example0.jira.com",
  username: "adminUser",
  password: "abc123",
  project_key: "EXAMPLE",
  enable_software_vulnerabilities: false,
};

const mockJira1 = {
  url: "https://example1.jira.com",
  username: "adminUser",
  password: "abc123",
  project_key: "PROJECT",
  enable_software_vulnerabilities: false,
};

const mockJira2 = {
  url: "https://example2.jira.com",
  username: "adminUser",
  password: "abc123",
  project_key: "KEY",
  enable_software_vulnerabilities: true,
};

const mockIntegration1 = {
  jira: [mockJira0, mockJira1],
};

const mockIntegration2 = {
  jira: [mockJira2],
};

const mockIntegrationAdd2 = {
  jira: [mockJira0, mockJira1, mockJira2],
};

const mockConfig = {
  org_info: {
    org_name: "s",
    org_logo_url: "",
  },
  server_settings: {
    server_url: "https://localhost:8080",
    live_query_disabled: false,
    enable_analytics: true,
    deferred_save_host: false,
  },
  smtp_settings: {
    enable_smtp: false,
    configured: true,
    sender_address: "",
    server: "",
    port: 0,
    authentication_type: "authtype_none",
    user_name: "",
    password: "",
    enable_ssl_tls: true,
    authentication_method: "authmethod_plain",
    domain: "",
    verify_ssl_certs: true,
    enable_start_tls: true,
  },
  host_expiry_settings: {
    host_expiry_enabled: true,
    host_expiry_window: 9,
  },
  host_settings: {
    enable_host_users: true,
    enable_software_inventory: true,
  },
  agent_options: {
    config: {
      options: {
        logger_plugin: "tls",
        pack_delimiter: "/",
        logger_tls_period: 100,
        distributed_plugin: "tls",
        disable_distributed: false,
        logger_tls_endpoint: "/api/v1/osquery/log",
        distributed_interval: 10,
        distributed_tls_max_attempts: 3,
      },
      decorators: {
        load: [
          "SELECT uuid AS host_uuid FROM system_info;",
          "SELECT hostname AS hostname FROM system_info;",
        ],
      },
    },
    overrides: {},
  },
  sso_settings: {
    entity_id: "",
    issuer_uri: "",
    idp_image_url: "",
    metadata: "",
    metadata_url: "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
    idp_name: "",
    enable_sso: false,
    enable_sso_idp_login: false,
  },
  vulnerability_settings: {
    databases_path: "",
  },
  webhook_settings: {
    host_status_webhook: {
      enable_host_status_webhook: false,
      destination_url: "",
      host_percentage: 0,
      days_count: 0,
    },
    failing_policies_webhook: {
      enable_failing_policies_webhook: false,
      destination_url: "",
      policy_ids: [],
      host_batch_size: 0,
    },
    vulnerabilities_webhook: {
      enable_vulnerabilities_webhook: true,
      destination_url: "www.example.com/",
      host_batch_size: 0,
    },
    interval: "24h0m0s",
  },
  update_interval: {
    osquery_detail: 10000000000,
    osquery_policy: 3600000000000,
  },
  vulnerabilities: {
    databases_path: "/tmp/vulndbs",
    periodicity: 3600000000000,
    cpe_database_url: "",
    cve_feed_prefix_url: "",
    current_instance_checks: "auto",
    disable_data_sync: false,
  },
  license: {
    tier: "premium",
    organization: "development-only",
    device_count: 100,
    expiration: "2022-06-30T20:00:00-04:00",
    note: "for development only",
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
};

export default {
  config1: { ...mockConfig, integrations: mockIntegration1 },
  config2: {
    ...mockConfig,
    integrations: mockIntegration2,
  },
  configAdd2: {
    ...mockConfig,
    integrations: mockIntegrationAdd2,
  },
};
