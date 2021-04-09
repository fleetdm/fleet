import PropTypes from "prop-types";

export default PropTypes.shape({
  admin: PropTypes.bool,
  email: PropTypes.string,
  gravatarURL: PropTypes.string,
  id: PropTypes.number,
  invited_by: PropTypes.number,
  name: PropTypes.string,
});
