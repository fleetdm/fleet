import PropTypes from "prop-types";

export default PropTypes.shape({
  live_query_disabled: PropTypes.bool,
  authentication_method: PropTypes.string,
  authentication_type: PropTypes.string,
  agent_options: PropTypes.string,
  configured: PropTypes.bool,
  domain: PropTypes.string,
  enable_analytics: PropTypes.bool,
  enable_ssl_tls: PropTypes.bool,
  enabled_sso: PropTypes.bool,
  enable_start_tls: PropTypes.bool,
  host_expiry_enabled: PropTypes.bool,
  host_expiry_window: PropTypes.number,
  server_url: PropTypes.string,
  org_logo_url: PropTypes.string,
  org_name: PropTypes.string,
  password: PropTypes.string,
  port: PropTypes.number,
  sender_address: PropTypes.string,
  server: PropTypes.string,
  user_name: PropTypes.string,
  verify_sll_certs: PropTypes.bool,
  tier: PropTypes.string,
});

export interface IConfig {
  live_query_disabled: boolean;
  authentication_method: string;
  authentication_type: string;
  agent_options: string;
  configured: boolean;
  domain: string;
  enable_analytics: boolean;
  enable_ssl_tls: boolean;
  enable_sso: boolean;
  enable_start_tls: boolean;
  host_expiry_enabled: boolean;
  host_expiry_window: number;
  server_url: string;
  org_logo_url: string;
  org_name: string;
  password: string;
  port: number;
  sender_address: string;
  server: string;
  user_name: string;
  verify_sll_certs: boolean;
  tier: string;
}
