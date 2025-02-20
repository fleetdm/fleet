import { IConfig, IMdmConfig } from "interfaces/config";

const DEFAULT_CONFIG_MDM_MOCK: IMdmConfig = {
  apple_server_url: "",
  enable_disk_encryption: false,
  windows_enabled_and_configured: true,
  apple_bm_default_team: "Apples",
  apple_bm_enabled_and_configured: true,
  apple_bm_terms_expired: false,
  enabled_and_configured: true,
  macos_updates: {
    minimum_version: "",
    deadline: "",
  },
  ios_updates: {
    minimum_version: "",
    deadline: "",
  },
  ipados_updates: {
    minimum_version: "",
    deadline: "",
  },
  macos_settings: {
    custom_settings: null,
    enable_disk_encryption: false,
  },
  macos_setup: {
    bootstrap_package: "",
    enable_end_user_authentication: false,
    macos_setup_assistant: null,
    enable_release_device_manually: false,
  },
  macos_migration: {
    enable: false,
    mode: "",
    webhook_url: "",
  },
  windows_updates: {
    deadline_days: null,
    grace_period_days: null,
  },
  windows_migration_enabled: false,
  end_user_authentication: {
    entity_id: "",
    issuer_uri: "",
    metadata: "",
    metadata_url: "",
    idp_name: "",
  },
};

export const createMockMdmConfig = (
  overrides?: Partial<IMdmConfig>
): IMdmConfig => {
  return { ...DEFAULT_CONFIG_MDM_MOCK, ...overrides };
};

const DEFAULT_CONFIG_MOCK: IConfig = {
  org_info: {
    org_name: "fleet",
    org_logo_url: "",
    org_logo_url_light_background: "",
    contact_url: "https://fleetdm.com/company/contact",
  },
  server_settings: {
    server_url: "https://localhost:8080",
    live_query_disabled: false,
    enable_analytics: true,
    deferred_save_host: false,
    query_reports_disabled: false,
    scripts_disabled: false,
    ai_features_disabled: false,
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
    issuer_uri: "",
    metadata: "",
    metadata_url: "",
    idp_name: "",
    idp_image_url: "",
    enable_sso: false,
    enable_sso_idp_login: false,
    enable_jit_provisioning: false,
    enable_jit_role_sync: false,
  },
  host_expiry_settings: {
    host_expiry_enabled: false,
    host_expiry_window: 0,
  },
  activity_expiry_settings: {
    activity_expiry_enabled: true,
    activity_expiry_window: 90,
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
    activities_webhook: {
      enable_activities_webhook: true,
      destination_url: "https://server.com",
    },
  },
  integrations: {
    jira: [],
    zendesk: [],
    google_calendar: [],
    ndes_scep_proxy: null,
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
    audit: {
      plugin: "",
      config: null,
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
  sandbox_enabled: false,
  features: {
    enable_host_users: true,
    enable_software_inventory: true,
  },
  fleet_desktop: { transparency_url: "https://fleetdm.com/transparency" },
  mdm: createMockMdmConfig(),
};

const createMockConfig = (overrides?: Partial<IConfig>): IConfig => {
  return { ...DEFAULT_CONFIG_MOCK, ...overrides };
};

export default createMockConfig;
