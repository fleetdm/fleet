import PropTypes from "prop-types";

export default PropTypes.shape({
  id: PropTypes.number,
  type: PropTypes.string,
  interval: PropTypes.number,
  query: PropTypes.string,
  built_in: PropTypes.bool,
  updated_at: PropTypes.string,
});
