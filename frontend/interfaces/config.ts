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
  issuer_uri: PropTypes.string,
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
});

export interface IFleetDesktopSettings {
  transparency_url: string;
}

export interface IConfigFormData {
  smtpAuthenticationMethod: string;
  smtpAuthenticationType: string;
  domain: string;
  smtpEnableSSLTLS: boolean;
  enableStartTLS: boolean;
  serverURL: string;
  orgLogoURL: string;
  orgName: string;
  smtpPassword: string;
  smtpPort?: number;
  smtpSenderAddress: string;
  smtpServer: string;
  smtpUsername: string;
  verifySSLCerts: boolean;
  entityID: string;
  issuerURI: string;
  idpImageURL: string;
  metadata: string;
  metadataURL: string;
  idpName: string;
  enableSSO: boolean;
  enableSSOIDPLogin: boolean;
  enableSMTP: boolean;
  enableHostExpiry: boolean;
  hostExpiryWindow: number;
  disableLiveQuery: boolean;
  agentOptions: any;
  enableHostStatusWebhook: boolean;
  hostStatusWebhookDestinationURL?: string;
  hostStatusWebhookHostPercentage?: number;
  hostStatusWebhookDaysCount?: number;
  enableUsageStatistics: boolean;
  transparency_url: string;
}

export interface IConfig {
  org_info: {
    org_name: string;
    org_logo_url: string;
  };
  sandbox_enabled: boolean;
  server_settings: {
    server_url: string;
    live_query_disabled: boolean;
    enable_analytics: boolean;
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
  };
  host_expiry_settings: {
    host_expiry_enabled: boolean;
    host_expiry_window: number;
  };
  host_settings: {
    enable_host_users: boolean;
    enable_software_inventory: boolean;
  };
  agent_options: string;
  update_interval: {
    osquery_detail: number;
    osquery_policy: number;
  };
  license: {
    organization: string;
    device_count: number;
    tier: string;
    expiration: string;
    note: string;
  };
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
  };
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
