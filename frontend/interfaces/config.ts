/* Config interface is a flattened version of the fleet/config API response */
import {
  IWebhookHostStatus,
  IWebhookFailingPolicies,
  IWebhookSoftwareVulnerabilities,
  IWebhookActivities,
} from "interfaces/webhook";
import { IGlobalIntegrations } from "./integration";

export interface ILicense {
  tier: string;
  device_count: number;
  expiration: string;
  note: string;
  organization: string;
}

export interface IEndUserAuthentication {
  entity_id: string;
  idp_name: string;
  issuer_uri: string;
  metadata: string;
  metadata_url: string;
}

export interface IMacOsMigrationSettings {
  enable: boolean;
  mode: "voluntary" | "forced" | "";
  webhook_url: string;
}

interface ICustomSetting {
  path: string;
  labels_include_all?: string[];
  labels_exclude_any?: string[];
}

export interface IAppleDeviceUpdates {
  minimum_version: string;
  deadline: string;
}

export interface IMdmConfig {
  /** Update this URL if you're self-hosting Fleet and you want your hosts to talk to a different URL for MDM features. (If not configured, hosts will use the base URL of the Fleet instance.) */
  apple_server_url: string;
  enable_disk_encryption: boolean;
  /** `enabled_and_configured` only tells us if Apples MDM has been enabled and
  configured correctly. The naming is slightly confusing but at one point we
  only supported apple mdm, so thats why it's name the way it is. */
  enabled_and_configured: boolean;
  apple_bm_default_team?: string;
  /**
   * @deprecated
   * Refer to needsAbmTermsRenewal from AppContext instead of config.apple_bm_terms_expired.
   * https://github.com/fleetdm/fleet/pull/21043/files#r1705977965
   */
  apple_bm_terms_expired: boolean;
  apple_bm_enabled_and_configured: boolean;
  windows_enabled_and_configured: boolean;
  windows_migration_enabled: boolean;
  end_user_authentication: IEndUserAuthentication;
  macos_updates: IAppleDeviceUpdates;
  ios_updates: IAppleDeviceUpdates;
  ipados_updates: IAppleDeviceUpdates;
  macos_settings: {
    custom_settings: null | ICustomSetting[];
    enable_disk_encryption: boolean;
  };
  macos_setup: {
    bootstrap_package: string | null;
    enable_end_user_authentication: boolean;
    macos_setup_assistant: string | null;
    enable_release_device_manually: boolean | null;
  };
  macos_migration: IMacOsMigrationSettings;
  windows_updates: {
    deadline_days: number | null;
    grace_period_days: number | null;
  };
}

// Note: IDeviceGlobalConfig is misnamed on the backend because in some cases it returns team config
// values if the device is assigned to a team, e.g., features.enable_software_inventory reflects the
// team config, if applicable, rather than the global config.
export interface IDeviceGlobalConfig {
  mdm: Pick<IMdmConfig, "enabled_and_configured">;
  features: Pick<IConfigFeatures, "enable_software_inventory">;
}

export interface IFleetDesktopSettings {
  transparency_url: string;
}

export interface IConfigFeatures {
  enable_host_users: boolean;
  enable_software_inventory: boolean;
}

export interface IConfigServerSettings {
  server_url: string;
  live_query_disabled: boolean;
  enable_analytics: boolean;
  deferred_save_host: boolean;
  query_reports_disabled: boolean;
  scripts_disabled: boolean;
  ai_features_disabled: boolean;
}

export interface IConfig {
  org_info: {
    org_name: string;
    org_logo_url: string;
    org_logo_url_light_background: string;
    contact_url: string;
  };
  sandbox_enabled: boolean;
  server_settings: IConfigServerSettings;
  smtp_settings?: {
    enable_smtp: boolean;
    configured?: boolean;
    sender_address: string;
    server: string;
    port?: number;
    authentication_type: string;
    user_name: string;
    password: string;
    enable_ssl_tls: boolean;
    authentication_method: string;
    domain: string;
    verify_ssl_certs: boolean;
    enable_start_tls: boolean;
  };
  sso_settings: {
    entity_id: string;
    issuer_uri: string;
    idp_image_url: string;
    metadata: string;
    metadata_url: string;
    idp_name: string;
    enable_sso: boolean;
    enable_sso_idp_login: boolean;
    enable_jit_provisioning: boolean;
    enable_jit_role_sync: boolean;
  };
  host_expiry_settings: {
    host_expiry_enabled: boolean;
    host_expiry_window?: number;
  };
  activity_expiry_settings: {
    activity_expiry_enabled: boolean;
    activity_expiry_window?: number;
  };
  features: IConfigFeatures;
  agent_options: unknown; // Can pass empty object
  update_interval: {
    osquery_detail: number;
    osquery_policy: number;
  };
  license: ILicense;
  fleet_desktop: IFleetDesktopSettings;
  vulnerabilities: {
    databases_path: string;
    periodicity: number;
    cpe_database_url: string;
    cve_feed_prefix_url: string;
    current_instance_checks: string;
    disable_data_sync: boolean;
    recent_vulnerability_max_age: number;
  };
  // Note: `vulnerability_settings` is deprecated and should not be used
  // vulnerability_settings: {
  //   databases_path: string;
  // };
  webhook_settings: IWebhookSettings;
  integrations: IGlobalIntegrations;
  logging: {
    debug: boolean;
    json: boolean;
    result: {
      plugin: string;
      config: {
        status_log_file: string;
        result_log_file: string;
        enable_log_rotation: boolean;
        enable_log_compression: boolean;
      };
    };
    status: {
      plugin: string;
      config: {
        status_log_file: string;
        result_log_file: string;
        enable_log_rotation: boolean;
        enable_log_compression: boolean;
      };
    };
    audit?: {
      plugin: string;
      config: any;
    };
  };
  email?: {
    backend: string;
    config: {
      region: string;
      source_arn: string;
    };
  };
  mdm: IMdmConfig;
}

export interface IWebhookSettings {
  failing_policies_webhook: IWebhookFailingPolicies;
  host_status_webhook: IWebhookHostStatus | null;
  vulnerabilities_webhook: IWebhookSoftwareVulnerabilities;
  activities_webhook: IWebhookActivities;
}

export type IAutomationsConfig = Pick<
  IConfig,
  "webhook_settings" | "integrations"
>;

export const CONFIG_DEFAULT_RECENT_VULNERABILITY_MAX_AGE_IN_DAYS = 30;

export interface IUserSettings {
  hidden_host_columns: string[];
}
