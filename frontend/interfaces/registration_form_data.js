import PropTypes from "prop-types";

export default PropTypes.shape({
  name: PropTypes.string,
  password: PropTypes.string,
  password_confirmation: PropTypes.string,
  email: PropTypes.string,
  org_name: PropTypes.string,
  org_web_url: PropTypes.string,
  org_logo_url: PropTypes.string,
  fleet_web_address: PropTypes.string,
});
