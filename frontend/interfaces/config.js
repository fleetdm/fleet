import PropTypes from "prop-types";

export default PropTypes.shape({
  live_query_disabled: PropTypes.bool,
  authentication_method: PropTypes.string,
  authentication_type: PropTypes.string,
  configured: PropTypes.bool,
  domain: PropTypes.string,
  enable_ssl_tls: PropTypes.bool,
  enable_start_tls: PropTypes.bool,
  host_expiry_enabled: PropTypes.bool,
  host_expiry_window: PropTypes.number,
  kolide_server_url: PropTypes.string,
  org_logo_url: PropTypes.string,
  org_name: PropTypes.string,
  password: PropTypes.string,
  port: PropTypes.number,
  sender_address: PropTypes.string,
  server: PropTypes.string,
  user_name: PropTypes.string,
  verify_sll_certs: PropTypes.bool,
});
