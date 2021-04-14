import PropTypes from "prop-types";

export default PropTypes.shape({
  type: PropTypes.string,
  name: PropTypes.string,
  installed_version: PropTypes.string,
  id: PropTypes.number,
});
