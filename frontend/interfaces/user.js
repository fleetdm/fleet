import PropTypes from "prop-types";

export default PropTypes.shape({
  admin: PropTypes.bool,
  email: PropTypes.string,
  enabled: PropTypes.bool,
  force_password_reset: PropTypes.bool,
  gravatarURL: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  position: PropTypes.string,
  username: PropTypes.string,
});
