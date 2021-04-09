import PropTypes from "prop-types";

export default PropTypes.shape({
  allowed_hosts: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
  expiry: PropTypes.string,
  hosts: PropTypes.number,
});
