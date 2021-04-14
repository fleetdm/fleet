import PropTypes from "prop-types";

export default PropTypes.shape({
  username: PropTypes.string,
  password: PropTypes.string,
  password_confirmation: PropTypes.string,
  email: PropTypes.string,
  org_name: PropTypes.string,
  org_web_url: PropTypes.string,
  org_logo_url: PropTypes.string,
  kolide_web_address: PropTypes.string,
});
