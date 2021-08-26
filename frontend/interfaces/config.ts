/* Config interface is a flattened version of the fleet/config API response */

import PropTypes from "prop-types";

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
  idp_name: PropTypes.string,
  enable_sso: PropTypes.bool,
  enable_sso_idp_login: PropTypes.bool,
  host_expiry_enabled: PropTypes.bool,
  host_expiry_window: PropTypes.number,
  agent_options: PropTypes.string,
  tier: PropTypes.string,
  organization: PropTypes.string,
  device_count: PropTypes.number,
  expiration: PropTypes.string,
  note: PropTypes.string,
  // vulnerability_settings: PropTypes.any, TODO
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

export interface IConfig {
  org_name: string;
  org_logo_url: string;
  server_url: string;
  live_query_disabled: boolean;
  enable_analytics: boolean;
  enable_smtp: boolean;
  configured: boolean;
  sender_address: string;
  server: string;
  port: number;
  authentication_type: string;
  user_name: string;
  password: string;
  enable_ssl_tls: boolean;
  authentication_method: string;
  domain: string;
  verify_sll_certs: boolean;
  enable_start_tls: boolean;
  entity_id: string;
  issuer_uri: string;
  idp_image_url: string;
  metadata: string;
  idp_name: string;
  enable_sso: boolean;
  enable_sso_idp_login: boolean;
  host_expiry_enabled: boolean;
  host_expiry_window: number;
  agent_options: string;
  tier: string;
  organization: string;
  device_count: number;
  expiration: string;
  note: string;
  // vulnerability_settings: any; TODO
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
