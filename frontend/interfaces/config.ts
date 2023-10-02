/* Config interface is a flattened version of the fleet/config API response */

import {
  IWebhookHostStatus,
  IWebhookFailingPolicies,
  IWebhookSoftwareVulnerabilities,
} from "interfaces/webhook";
import PropTypes from "prop-types";
import { IIntegrations } from "./integration";

export default PropTypes.shape({
  org_name: PropTypes.string,
  org_logo_url: PropTypes.string,
  contact_url: PropTypes.string,
  server_url: PropTypes.string,
  live_query_disabled: PropTypes.bool,
  enable_analytics: PropTypes.bool,
  enable_smtp: PropTypes.bool,
  configured: PropTypes.bool,
  sender_address: PropTypes.string,
  server: PropTypes.string,
  port: PropTypes.number,
  authentication_type: PropTypes.string,
  user_name: PropTypes.string,
  password: PropTypes.string,
  enable_ssl_tls: PropTypes.bool,
  authentication_method: PropTypes.string,
  domain: PropTypes.string,
  verify_sll_certs: PropTypes.bool,
  enable_start_tls: PropTypes.bool,
  entity_id: PropTypes.string,
  idp_image_url: PropTypes.string,
  metadata: PropTypes.string,
  metadata_url: PropTypes.string,
  idp_name: PropTypes.string,
  enable_sso: PropTypes.bool,
  enable_sso_idp_login: PropTypes.bool,
  enable_jit_provisioning: PropTypes.bool,
  host_expiry_enabled: PropTypes.bool,
  host_expiry_window: PropTypes.number,
  agent_options: PropTypes.string,
  tier: PropTypes.string,
  organization: PropTypes.string,
  device_count: PropTypes.number,
  expiration: PropTypes.string,
  mdm: PropTypes.shape({
    enabled_and_configured: PropTypes.bool,
    apple_bm_terms_expired: PropTypes.bool,
    apple_bm_enabled_and_configured: PropTypes.bool,
    windows_enabled_and_configured: PropTypes.bool,
    macos_updates: PropTypes.shape({
      minimum_version: PropTypes.string,
      deadline: PropTypes.string,
    }),
  }),
  note: PropTypes.string,
  // vulnerability_settings: PropTypes.any, TODO
  enable_host_status_webhook: PropTypes.bool,
  destination_url: PropTypes.string,
  host_percentage: PropTypes.number,
  days_count: PropTypes.number,
  logging: PropTypes.shape({
    debug: PropTypes.bool,
    json: PropTypes.bool,
    result: PropTypes.shape({
      plugin: PropTypes.string,
      config: PropTypes.shape({
        status_log_file: PropTypes.string,
        result_log_file: PropTypes.string,
        enable_log_rotation: PropTypes.bool,
        enable_log_compression: PropTypes.bool,
      }),
    }),
    status: PropTypes.shape({
      plugin: PropTypes.string,
      config: PropTypes.shape({
        status_log_file: PropTypes.string,
        result_log_file: PropTypes.string,
        enable_log_rotation: PropTypes.bool,
        enable_log_compression: PropTypes.bool,
      }),
    }),
  }),
  email: PropTypes.shape({
    backend: PropTypes.string,
    config: PropTypes.shape({
      region: PropTypes.string,
      source_arn: PropTypes.string,
    }),
  }),
});

export interface ILicense {
  tier: string;
  device_count: number;
  expiration: string;
  note: string;
  organization: string;
}

interface IEndUserAuthentication {
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

export interface IMdmConfig {
  enabled_and_configured: boolean;
  apple_bm_default_team?: string;
  apple_bm_terms_expired: boolean;
  apple_bm_enabled_and_configured: boolean;
  windows_enabled_and_configured: boolean;
  end_user_authentication: IEndUserAuthentication;
  macos_updates: {
    minimum_version: string;
    deadline: string;
  };
  macos_settings: {
    custom_settings: null;
    enable_disk_encryption: boolean;
  };
  macos_setup: {
    bootstrap_package: string | null;
    enable_end_user_authentication: boolean;
    macos_setup_assistant: string | null;
  };
  macos_migration: IMacOsMigrationSettings;
}

export interface IDeviceGlobalConfig {
  mdm: Pick<IMdmConfig, "enabled_and_configured">;
}

export interface IFleetDesktopSettings {
  transparency_url: string;
}

export interface IConfigFormData {
  smtpAuthenticationMethod: string;
  smtpAuthenticationType: string;
  domain: string;
  smtpEnableSslTls: boolean;
  enableStartTls: boolean;
  serverUrl: string;
  orgLogoUrl: string;
  orgName: string;
  smtpPassword: string;
  smtpPort?: number;
  smtpSenderAddress: string;
  smtpServer: string;
  smtpUsername: string;
  verifySslCerts: boolean;
  entityId: string;
  idpImageUrl: string;
  metadata: string;
  metadataUrl: string;
  idpName: string;
  enableSso: boolean;
  enableSsoIdpLogin: boolean;
  enableSmtp: boolean;
  enableHostExpiry: boolean;
  hostExpiryWindow: number;
  disableLiveQuery: boolean;
  agentOptions: any;
  enableHostStatusWebhook: boolean;
  hostStatusWebhookDestinationUrl?: string;
  hostStatusWebhookHostPercentage?: number;
  hostStatusWebhookDaysCount?: number;
  enableUsageStatistics: boolean;
  transparencyUrl: string;
}

export interface IConfigFeatures {
  enable_host_users: boolean;
  enable_software_inventory: boolean;
}

export interface IConfig {
  org_info: {
    org_name: string;
    org_logo_url: string;
    org_logo_url_light_background: string;
    contact_url: string;
  };
  sandbox_enabled: boolean;
  server_settings: {
    server_url: string;
    live_query_disabled: boolean;
    enable_analytics: boolean;
    deferred_save_host: boolean;
    query_reports_disabled: boolean;
  };
  smtp_settings: {
    enable_smtp: boolean;
    configured: boolean;
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
    host_expiry_window: number;
  };
  features: IConfigFeatures;
  agent_options: string;
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
  integrations: IIntegrations;
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
  mdm_enabled?: boolean; // TODO: remove when windows MDM is released. Only used for windows MDM dev currently.
}

export interface IWebhookSettings {
  failing_policies_webhook: IWebhookFailingPolicies;
  host_status_webhook: IWebhookHostStatus;
  vulnerabilities_webhook: IWebhookSoftwareVulnerabilities;
}

export type IAutomationsConfig = Pick<
  IConfig,
  "webhook_settings" | "integrations"
>;

export const CONFIG_DEFAULT_RECENT_VULNERABILITY_MAX_AGE_IN_DAYS = 30;
